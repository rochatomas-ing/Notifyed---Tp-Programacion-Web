---
name: aprender-opencode
description: Mini-referencia de conceptos de vibecoding, opencode y Claude Code (AGENTS.md, agents, skills, comandos, permisos, MCP). Usar cuando el usuario quiera aprender o entender conceptos de estas herramientas, o pregunte "qué es un agent/skill", "cómo configuro X", "cómo funciona el vibecoding".
---

# Aprender opencode / vibecoding

Mini-referencia para orientar al tutor y al usuario. Combinar con el agente `tutor` para explicaciones guiadas.

## Vibecoding

- **Qué es**: estilo de desarrollo donde describís la intención (en lenguaje natural) y un agente de IA escribe la mayor parte del código. Vos dirigís y revisás.
- **Clave**: es un dialogo de ida y vuelta. No es "copy-paste mágico": cuanto mejor describes el contexto, mejor el resultado.
- **Contexto**: la calidad de la salida depende del contexto que le des al agente — por eso existen `AGENTS.md`, skills y agents.

## Los conceptos (y dónde viven en este repo)

| Concepto | Qué es | Dónde está acá |
| --- | --- | --- |
| `AGENTS.md` | Contexto del proyecto que cualquier agente lee antes de tocar código | raíz del repo |
| `opencode.json` | Config del proyecto (permisos, instructions, modelo) | `.opencode/opencode.json` |
| **Agent** | Un "rol" con prompt + permisos propios (primary o subagent) | `.opencode/agent/*.md` |
| **Skill** | Guía temática que se carga cuando el tema aparece (skill de "recetas SQL", "recetas Docker"...) | `.opencode/skill/<nombre>/SKILL.md` |
| **Command** | Atajo `/comando` que dispara un prompt | `.opencode/command/*.md` |
| **Permission** | Qué puede hacer el agente sin preguntar (edit, bash, read...) | dentro de `opencode.json` o del agent |
| **MCP** | Servidores externos que le dan herramientas nuevas al agente (datos, navegador, etc.) | `mcp` en `opencode.json` (todavía no configurado acá) |

## Modelo mental de un agent

`AGENTS.md` le dice al agente **el contexto**; el `prompt` del agent le dice **el rol**; las `permissions` le dicen **qué puede hacer**; los `skills` le dicen **qué procedimiento seguir cuando aparece un tema**.

## Regla de oro de los permisos

El orden importa: opencode usa la **última regla que matchea**. Reglas amplias primero (`"*": "ask"`), reglas específicas después (`"go *": "allow"`).

## Comandos útiles de opencode

- `/help` — ayuda de opencode.
- `/aprender <tema>` — invoca al agente tutor (creado en este repo).
- `/test` — corre el flujo de tests del repo.
- `opencode auth list` — qué proveedores de IA tenés conectados.

## Para profundizar

- `https://opencode.ai/docs` — documentación oficial de opencode.
- `https://opencode.ai/config.json` — esquema válido de la config (opencode falla si un campo está mal).