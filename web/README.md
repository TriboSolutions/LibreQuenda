# Web de LibreQuenda

A aplicación web de LibreQuenda, feita con React, TypeScript e Vite: a veciñanza reserva desde ela, o concello xestiona en `/<concello>/admin` e a superadministración en `/admin`. Os textos están en `src/locales` (gl, es, ca, en).

## Desenvolvemento

Precisa Node.js e pnpm (a versión está fixada en `packageManager` de `package.json`). Desde este directorio:

```sh
pnpm install
pnpm dev
```

O servidor de desenvolvemento de Vite reenvía `/api` e `/config.json` ao Caddy da pila de Docker Compose (`http://localhost` por defecto, ou o valor de `VITE_API_PROXY_TARGET`), así que antes cómpre ter arrincada a pila de desenvolvemento de LibreQuenda.

## Ordes

| Orde | Que fai |
|---|---|
| `pnpm dev` | servidor de desenvolvemento con recarga en quente |
| `pnpm build` | comproba os tipos (`tsc -b`) e compila para produción en `dist/` |
| `pnpm preview` | serve a compilación de `dist/` |
| `pnpm lint` | analiza o código con Oxlint |
| `pnpm test:a11y` | probas de accesibilidade (WCAG 2.1) con Playwright en Chromium, coa API simulada (`e2e/`) |

Para as probas de accesibilidade, instala antes o navegador con `pnpm exec playwright install --with-deps chromium`. A CI executa `pnpm lint`, `pnpm build` e `pnpm test:a11y`.

A pila, a estrutura do repositorio e o resto da documentación de desenvolvemento están en [docs/development.md](../docs/development.md).
