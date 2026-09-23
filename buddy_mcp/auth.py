"""OAuth token verification and authenticated subject lookup."""

from __future__ import annotations

import jwt
from mcp.server.auth.middleware.auth_context import get_access_token
from mcp.server.auth.provider import AccessToken, TokenVerifier


class BuddyTokenVerifier(TokenVerifier):
    def __init__(self, issuer: str, resource: str) -> None:
        self.issuer = issuer
        self.resource = resource
        self.jwks = jwt.PyJWKClient(f"{issuer}/.well-known/jwks.json", cache_jwk_set=True)

    async def verify_token(self, token: str) -> AccessToken | None:
        try:
            key = self.jwks.get_signing_key_from_jwt(token).key
            claims = jwt.decode(
                token,
                key,
                algorithms=["RS256"],
                issuer=self.issuer,
                audience=self.resource,
                options={"require": ["exp", "iss", "aud", "sub"]},
            )
            scopes = claims.get("scope", "")
            if isinstance(scopes, str):
                scopes = scopes.split()
            if "tools:read" not in scopes:
                return None
            return AccessToken(
                token=token,
                client_id=str(claims.get("client_id", claims.get("azp", "oauth-client"))),
                scopes=list(scopes),
                expires_at=int(claims["exp"]),
                resource=self.resource,
                subject=str(claims["sub"]),
                claims={"iss": self.issuer},
            )
        except (jwt.PyJWTError, ValueError, KeyError, TypeError):
            return None


def current_owner() -> str:
    token = get_access_token()
    if token is None or not token.subject:
        raise PermissionError("Buddy requires an authenticated OAuth user")
    return str(token.subject)
