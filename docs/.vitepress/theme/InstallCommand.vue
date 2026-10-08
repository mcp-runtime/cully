<script setup lang="ts">
import { computed, ref } from 'vue'
import { agents, selectedAgent as selected } from './agent'

const props = defineProps<{ kind?: 'install' | 'setup'; template?: string }>()
const copied = ref(false)
const command = computed(() => {
  if (props.template) return props.template.replace('{agent}', selected.value)
  return props.kind === 'setup'
    ? `cully setup --agent ${selected.value} --all`
    : `curl -fsSL https://cully.net/install.sh | sh -s -- --agent ${selected.value}`
})

async function copy() {
  try {
    await navigator.clipboard.writeText(command.value)
    copied.value = true
    setTimeout(() => (copied.value = false), 1600)
  } catch {
    copied.value = false
  }
}
</script>

<template>
  <div class="install-window">
    <div class="install-bar">
      <span class="install-dots" aria-hidden="true">
        <i class="dot red"></i><i class="dot yellow"></i><i class="dot green"></i>
      </span>
      <div class="install-tabs" role="tablist" aria-label="Coding agent">
        <button
          v-for="agent in agents"
          :key="agent.id"
          role="tab"
          type="button"
          :aria-selected="selected === agent.id"
          :class="{ active: selected === agent.id }"
          @click="selected = agent.id"
        >
          {{ agent.label }}
        </button>
      </div>
    </div>
    <div class="install-body">
      <code><span class="prompt">$</span> {{ command }}</code>
      <button type="button" class="install-copy" @click="copy">
        {{ copied ? 'Copied' : 'Copy' }}
      </button>
    </div>
    <div v-if="!kind && !template" class="install-guides">
      <span>Installation guide</span>
      <a href="/installation#macos">macOS</a>
      <a href="/installation#linux">Linux</a>
    </div>
  </div>
</template>

<style scoped>
.install-window {
  margin: 16px 0;
  border: 1px solid var(--vp-c-divider);
  border-radius: 10px;
  overflow: hidden;
  background: var(--vp-c-bg-soft);
}
.install-bar {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 10px 14px;
  border-bottom: 1px solid var(--vp-c-divider);
  background: var(--vp-c-bg-alt);
}
.install-dots {
  display: flex;
  gap: 7px;
}
.dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  display: block;
}
.red { background: #ff5f57; }
.yellow { background: #febc2e; }
.green { background: #28c840; }
.install-tabs {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}
.install-tabs button {
  padding: 3px 12px;
  border-radius: 6px;
  border: 1px solid transparent;
  background: transparent;
  color: var(--vp-c-text-2);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
}
.install-tabs button:hover { color: var(--vp-c-text-1); }
.install-tabs button.active {
  background: var(--vp-c-brand-soft);
  border-color: var(--vp-c-brand-1);
  color: var(--vp-c-brand-1);
}
.install-body {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 16px;
}
.install-body code {
  overflow-x: auto;
  white-space: nowrap;
  font-size: 13px;
  background: none;
  color: var(--vp-c-text-1);
}
.prompt { color: var(--vp-c-brand-1); margin-right: 6px; }
.install-copy {
  flex-shrink: 0;
  padding: 4px 12px;
  border-radius: 6px;
  border: 1px solid var(--vp-c-divider);
  background: var(--vp-c-bg);
  font-size: 12px;
  cursor: pointer;
}
.install-copy:hover { border-color: var(--vp-c-brand-1); }
.install-guides {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  padding: 10px 16px;
  border-top: 1px solid var(--vp-c-divider);
  font-size: 13px;
  color: var(--vp-c-text-2);
}
.install-guides span { margin-right: 4px; }
.install-guides a {
  padding: 2px 12px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 999px;
  background: var(--vp-c-bg);
  color: var(--vp-c-text-1);
  text-decoration: none;
}
.install-guides a:hover {
  border-color: var(--vp-c-brand-1);
  color: var(--vp-c-brand-1);
}
</style>
