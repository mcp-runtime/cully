#!/usr/bin/env python3
"""Buddy local life capture CLI."""

from __future__ import annotations

import argparse
import json
import plistlib
import sqlite3
import subprocess
import sys
from dataclasses import dataclass
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parent
DEFAULT_QUESTIONS = ROOT / "questions.json"
DEFAULT_MEMORY = ROOT / "memory" / "buddy_memory.db"
LAUNCHD_DIR = Path.home() / "Library" / "LaunchAgents"
IST = timezone(timedelta(hours=5, minutes=30), name="IST")


@dataclass(frozen=True)
class Prompt:
    key: str
    question: str
    category: str


def ist_now() -> str:
    return datetime.now(IST).isoformat(timespec="seconds")


def display_timestamp(value: str) -> str:
    """Render event timestamps in IST; legacy naive values are interpreted as UTC."""
    parsed = datetime.fromisoformat(value)
    if parsed.tzinfo is None:
        parsed = parsed.replace(tzinfo=timezone.utc)
    return parsed.astimezone(IST).isoformat(timespec="seconds")


def parse_timestamp(value: str | None) -> str:
    if not value:
        return ist_now()
    try:
        parsed = datetime.fromisoformat(value)
    except ValueError as exc:
        raise SystemExit(
            "Use ISO time for --at, like 2026-07-20T06:00:00+05:30"
        ) from exc
    if parsed.tzinfo is None:
        parsed = parsed.replace(tzinfo=IST)
    return parsed.astimezone(IST).isoformat(timespec="seconds")


def load_questions(path: Path, mode: str) -> list[Prompt]:
    payload = json.loads(path.read_text(encoding="utf-8"))
    prompts = payload.get(mode)
    if not isinstance(prompts, list):
        raise SystemExit(f"Unknown question mode: {mode}")
    return [
        Prompt(
            key=str(item["key"]),
            question=str(item["question"]),
            category=str(item.get("category", mode)),
        )
        for item in prompts
    ]


def append_event(memory_path: Path, event: dict[str, Any]) -> None:
    with connect_memory(memory_path) as conn:
        if event.get("type") == "checkin":
            timestamp = str(event["timestamp"])
            mode = str(event.get("mode", "checkin"))
            for key, value in event.get("answers", {}).items():
                insert_event(
                    conn,
                    timestamp=timestamp,
                    event_type="checkin",
                    category=str(key),
                    mode=mode,
                    text=str(value),
                    payload={"key": key},
                )
        else:
            insert_event(
                conn,
                timestamp=str(event["timestamp"]),
                event_type="log",
                category=str(event.get("category", "other")),
                mode=None,
                text=str(event.get("text", "")),
                payload=dict(event.get("payload", {})),
            )


def read_events(memory_path: Path) -> list[dict[str, Any]]:
    return read_sqlite_events(memory_path)


def connect_memory(memory_path: Path) -> sqlite3.Connection:
    memory_path.parent.mkdir(parents=True, exist_ok=True)
    conn = sqlite3.connect(memory_path)
    conn.row_factory = sqlite3.Row
    conn.execute("PRAGMA journal_mode=WAL")
    conn.execute("PRAGMA synchronous=NORMAL")
    conn.execute(
        """
        CREATE TABLE IF NOT EXISTS events (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            timestamp TEXT NOT NULL,
            type TEXT NOT NULL,
            category TEXT NOT NULL,
            mode TEXT,
            text TEXT NOT NULL,
            payload_json TEXT NOT NULL DEFAULT '{}'
        )
        """
    )
    conn.execute(
        "CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp)"
    )
    conn.execute(
        "CREATE INDEX IF NOT EXISTS idx_events_category_timestamp ON events(category, timestamp)"
    )
    try:
        conn.execute(
            """
            CREATE VIRTUAL TABLE IF NOT EXISTS events_fts USING fts5(
                text,
                category,
                content='events',
                content_rowid='id'
            )
            """
        )
    except sqlite3.OperationalError:
        conn.execute(
            """
            CREATE VIRTUAL TABLE IF NOT EXISTS events_fts USING fts4(
                text,
                category
            )
            """
        )
    return conn


def insert_event(
    conn: sqlite3.Connection,
    *,
    timestamp: str,
    event_type: str,
    category: str,
    mode: str | None,
    text: str,
    payload: dict[str, Any],
) -> None:
    cursor = conn.execute(
        """
        INSERT INTO events(timestamp, type, category, mode, text, payload_json)
        VALUES (?, ?, ?, ?, ?, ?)
        """,
        (timestamp, event_type, category, mode, text, json.dumps(payload, sort_keys=True)),
    )
    rowid = cursor.lastrowid
    conn.execute(
        "INSERT INTO events_fts(rowid, text, category) VALUES (?, ?, ?)",
        (rowid, text, category),
    )
    conn.commit()


def row_to_event(row: sqlite3.Row) -> dict[str, Any]:
    if row["type"] == "checkin":
        key = json.loads(row["payload_json"] or "{}").get("key", row["category"])
        return {
            "type": "checkin",
            "mode": row["mode"] or "checkin",
            "timestamp": display_timestamp(row["timestamp"]),
            "answers": {key: row["text"]},
        }
    return {
        "type": "log",
        "category": row["category"],
        "timestamp": display_timestamp(row["timestamp"]),
        "text": row["text"],
        "payload": json.loads(row["payload_json"] or "{}"),
    }


def read_sqlite_events(memory_path: Path) -> list[dict[str, Any]]:
    if not memory_path.exists():
        return []
    with connect_memory(memory_path) as conn:
        rows = conn.execute(
            """
            SELECT timestamp, type, category, mode, text, payload_json
            FROM events
            ORDER BY timestamp, id
            """
        ).fetchall()
    events = [row_to_event(row) for row in rows]
    events.sort(
        key=lambda event: datetime.fromisoformat(event["timestamp"]).astimezone(timezone.utc)
    )
    return events


def ask(args: argparse.Namespace) -> None:
    prompts = load_questions(args.questions, args.mode)
    answers: dict[str, str] = {}
    print(f"Buddy {args.mode} check-in. Press enter to skip anything.\n")
    for prompt in prompts:
        answer = input(f"{prompt.question}\n> ").strip()
        if answer:
            answers[prompt.key] = answer

    if not answers:
        print("No answers saved.")
        return

    append_event(
        args.memory,
        {
            "type": "checkin",
            "mode": args.mode,
            "timestamp": ist_now(),
            "answers": answers,
        },
    )
    print(f"Saved {len(answers)} answer(s) to {args.memory}")


def log(args: argparse.Namespace) -> None:
    append_event(
        args.memory,
        {
            "type": "log",
            "category": args.category,
            "timestamp": parse_timestamp(args.at),
            "text": args.text,
        },
    )
    print(f"Saved {args.category} log to {args.memory}")


def food(args: argparse.Namespace) -> None:
    pieces = [f"Food: {args.text}"]
    payload: dict[str, Any] = {}
    if args.calories is not None:
        pieces.append(f"calories={args.calories}")
        payload["calories"] = args.calories
    if args.protein is not None:
        pieces.append(f"protein_g={args.protein}")
        payload["protein_g"] = args.protein
    if args.carbs is not None:
        pieces.append(f"carbs_g={args.carbs}")
        payload["carbs_g"] = args.carbs
    if args.fat is not None:
        pieces.append(f"fat_g={args.fat}")
        payload["fat_g"] = args.fat
    if args.confidence:
        pieces.append(f"confidence={args.confidence}")
        payload["confidence"] = args.confidence
    if args.assumptions:
        pieces.append(f"assumptions={args.assumptions}")
        payload["assumptions"] = args.assumptions

    append_event(
        args.memory,
        {
            "type": "log",
            "category": "food",
            "timestamp": parse_timestamp(args.at),
            "text": "; ".join(pieces),
            "payload": payload,
        },
    )
    print(f"Saved structured food log to {args.memory}")


def summary(args: argparse.Namespace) -> None:
    events = read_events(args.memory)
    if args.limit:
        events = events[-args.limit :]
    if not events:
        print("No Buddy memory yet.")
        return

    for event in events:
        stamp = event.get("timestamp", "unknown time")
        if event.get("type") == "checkin":
            print(f"\n[{stamp}] {event.get('mode', 'checkin')}")
            for key, value in event.get("answers", {}).items():
                print(f"- {key}: {value}")
        else:
            print(f"\n[{stamp}] {event.get('category', 'log')}: {event.get('text', '')}")


def analytics(args: argparse.Namespace) -> None:
    events = read_events(args.memory)
    if args.limit:
        events = events[-args.limit :]
    if not events:
        print("No Buddy memory yet.")
        return

    counts: dict[str, int] = {}
    recent_by_category: dict[str, list[str]] = {}
    calories_by_day: dict[str, int] = {}
    for event in events:
        if event.get("type") == "checkin":
            for key, value in event.get("answers", {}).items():
                counts[key] = counts.get(key, 0) + 1
                recent_by_category.setdefault(key, []).append(str(value))
        else:
            category = str(event.get("category", "other"))
            counts[category] = counts.get(category, 0) + 1
            recent_by_category.setdefault(category, []).append(str(event.get("text", "")))
            payload = event.get("payload", {})
            if category == "food" and isinstance(payload, dict):
                calories = payload.get("calories")
                if isinstance(calories, int):
                    day = str(event.get("timestamp", "unknown"))[:10]
                    calories_by_day[day] = calories_by_day.get(day, 0) + calories

    print("Buddy analytics\n")
    print("Capture volume:")
    for category, count in sorted(counts.items(), key=lambda item: (-item[1], item[0])):
        print(f"- {category}: {count}")

    print("\nRecent signals:")
    for category in ["career", "fitness", "food", "water", "relationship", "finance", "reading", "mood"]:
        values = [value for value in recent_by_category.get(category, []) if value][-3:]
        if values:
            print(f"- {category}: {' | '.join(values)}")

    if calories_by_day:
        print("\nCalories by day:")
        for day, calories in sorted(calories_by_day.items(), reverse=True):
            print(f"- {day}: {calories} kcal")

    gaps = [
        category
        for category in ["career", "fitness", "relationship", "finance", "food", "water", "reading"]
        if counts.get(category, 0) == 0
    ]
    if gaps:
        print("\nMissing signals to ask next:")
        print("- " + ", ".join(gaps))


def search(args: argparse.Namespace) -> None:
    if not args.memory.exists():
        print("No Buddy memory yet.")
        return

    with connect_memory(args.memory) as conn:
        try:
            rows = search_rows(conn, args.query, args.limit)
        except sqlite3.OperationalError:
            quoted_query = '"' + args.query.replace('"', '""') + '"'
            rows = search_rows(conn, quoted_query, args.limit)

    if not rows:
        print("No matching Buddy memory.")
        return

    for row in rows:
        mode = f" ({row['mode']})" if row["mode"] else ""
        print(f"[{row['timestamp']}] {row['category']}{mode}: {row['text']}")


def search_rows(conn: sqlite3.Connection, query: str, limit: int) -> list[sqlite3.Row]:
    return conn.execute(
        """
        SELECT e.timestamp, e.type, e.category, e.mode, e.text
        FROM events_fts
        JOIN events e ON e.id = events_fts.rowid
        WHERE events_fts MATCH ?
        ORDER BY e.timestamp DESC
        LIMIT ?
        """,
        (query, limit),
    ).fetchall()


def notify(args: argparse.Namespace) -> None:
    message = "Buddy check-in: send food, water, money, reading, workout, career, or relationship data."
    if sys.platform == "darwin":
        subprocess.run(
            [
                "osascript",
                "-e",
                f'display notification "{message}" with title "Buddy"',
            ],
            check=False,
        )
    print(message)


def launchd_plist(label: str, argv: list[str], interval_seconds: int) -> dict[str, Any]:
    return {
        "Label": label,
        "ProgramArguments": argv,
        "StartInterval": interval_seconds,
        "RunAtLoad": False,
        "StandardOutPath": str(ROOT / "memory" / f"{label}.out.log"),
        "StandardErrorPath": str(ROOT / "memory" / f"{label}.err.log"),
        "WorkingDirectory": str(ROOT),
    }


def install_launchd(args: argparse.Namespace) -> None:
    if sys.platform != "darwin":
        raise SystemExit("launchd install is only supported on macOS.")

    launchd_specs = [
        (
            "com.buddy.life-reminder",
            [sys.executable, str(ROOT / "buddy.py"), "notify"],
            args.reminder_minutes * 60,
        ),
        (
            "com.buddy.daily-review",
            [sys.executable, str(ROOT / "buddy.py"), "ask", "--mode", "daily"],
            24 * 60 * 60,
        ),
    ]

    LAUNCHD_DIR.mkdir(parents=True, exist_ok=True)
    for label, argv, interval in launchd_specs:
        path = LAUNCHD_DIR / f"{label}.plist"
        with path.open("wb") as handle:
            plistlib.dump(launchd_plist(label, argv, interval), handle, sort_keys=False)
        subprocess.run(["launchctl", "unload", str(path)], check=False)
        subprocess.run(["launchctl", "load", str(path)], check=True)
        print(f"Installed {path}")


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Capture life logs into local memory.")
    parser.add_argument("--memory", type=Path, default=DEFAULT_MEMORY)
    parser.add_argument("--questions", type=Path, default=DEFAULT_QUESTIONS)

    sub = parser.add_subparsers(required=True)

    ask_parser = sub.add_parser("ask", help="Run an interactive check-in.")
    ask_parser.add_argument("--mode", default="quick", choices=["quick", "daily", "weekly"])
    ask_parser.set_defaults(func=ask)

    log_parser = sub.add_parser("log", help="Append a free-form memory event.")
    log_parser.add_argument(
        "category",
        choices=[
            "career",
            "fitness",
            "relationship",
            "finance",
            "food",
            "water",
            "reading",
            "mood",
            "other",
        ],
    )
    log_parser.add_argument("--at", help="Event time, like 2026-07-20T06:00:00+05:30")
    log_parser.add_argument("text")
    log_parser.set_defaults(func=log)

    food_parser = sub.add_parser("food", help="Append a structured food memory event.")
    food_parser.add_argument("--at", help="Event time, like 2026-07-20T06:00:00+05:30")
    food_parser.add_argument("--calories", type=int)
    food_parser.add_argument("--protein", type=float)
    food_parser.add_argument("--carbs", type=float)
    food_parser.add_argument("--fat", type=float)
    food_parser.add_argument("--confidence", choices=["low", "medium", "high"])
    food_parser.add_argument("--assumptions")
    food_parser.add_argument("text")
    food_parser.set_defaults(func=food)

    summary_parser = sub.add_parser("summary", help="Print recent memory.")
    summary_parser.add_argument("--limit", type=int, default=20)
    summary_parser.set_defaults(func=summary)

    analytics_parser = sub.add_parser("analytics", help="Analyze recent memory.")
    analytics_parser.add_argument("--limit", type=int, default=200)
    analytics_parser.set_defaults(func=analytics)

    search_parser = sub.add_parser("search", help="Search memory using full-text search.")
    search_parser.add_argument("query")
    search_parser.add_argument("--limit", type=int, default=20)
    search_parser.set_defaults(func=search)

    notify_parser = sub.add_parser("notify", help="Show a reminder notification.")
    notify_parser.set_defaults(func=notify)

    install_parser = sub.add_parser("install-launchd", help="Install macOS reminders.")
    install_parser.add_argument("--reminder-minutes", type=int, default=120)
    install_parser.set_defaults(func=install_launchd)

    return parser


def main() -> None:
    args = build_parser().parse_args()
    args.func(args)


if __name__ == "__main__":
    main()
