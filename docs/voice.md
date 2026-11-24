# Reservar falando (liña de voz)

A liña de voz deixa buscar instalacións e quendas libres, e reservar, falando en galego: desde a web (o icono do teléfono na barra superior, ou «Reserva por voz» baixo as quendas dunha instalación) ou, se a instalación o configura, por teléfono. É opcional e vai no servizo `voice/` (Python, [Pipecat](https://github.com/pipecat-ai/pipecat)).

Por defecto todo o que escoita e di funciona no propio servidor, con modelos abertos do [Proxecto Nós](https://nos.gal/gl/proxecto-nos) (Universidade de Santiago de Compostela). O modelo de linguaxe que decide que buscar e que dicir escóllese na configuración: un servizo comercial (Claude, de Anthropic, por defecto) ou un modelo de pesos abertos no teu propio servidor. Cada provedor é un adaptador: cambialo non cambia nada máis da plataforma.

## Como funciona

```
navegador (micrófono e altofalante)        teléfono
        │ WebSocket, mesmo dominio               │ Twilio Media Streams
        ▼                                        ▼
  Caddy /voice/* ──────────────► voice (FastAPI + Pipecat)
                                  │
                                  ├─ Silero VAD: cando fala a persoa e cando cala
                                  ├─ STT: voz a texto (Nós Whisper ou wav2vec2, no servidor)
                                  ├─ LLM: que dicir, con ferramentas ──► API de LibreQuenda, /api/mcp
                                  ├─ normalización: "ás 18:30" -> "ás seis e media da tarde"
                                  └─ TTS: texto a voz (Nós, frase a frase)
```

- **Web:** o icono do teléfono na barra superior e a ligazón «Reserva por voz» baixo as quendas aparecen só cando a instalación ten a liña de voz acesa (`/voice/config.json` responde, como `/config.json` para o inicio de sesión), nas páxinas de busca e reserva. O panel pide o micrófono, mostra o estado da chamada, os **subtítulos en directo** das dúas partes (WCAG 1.2.4), silenciar e colgar; funciona co teclado e cun lector de pantalla. Coa sesión iniciada, a web envía o token de acceso da persoa nunha petición `POST /voice/session` e recibe un tícket dun só uso para abrir o WebSocket: o token nunca vai nun URL. O audio viaxa polo mesmo WebSocket do dominio (protobuf de Pipecat), sen TURN nin UDP.
- **Ferramentas:** o axente usa o mesmo [servidor MCP](mcp.md) que calquera outro axente, coas mesmas normas que a web. Nunha chamada web coa sesión iniciada, cada chamada a `/api/mcp` leva `Authorization: Bearer <token da persoa>`; sen sesión, só pode consultar (`search_facilities`, `facility_details`, `list_councils`, `list_events`). Nunha chamada de teléfono leva `X-Voice-Key` (a clave compartida coa API) e `X-Caller-Phone` (o número que chama), sempre na mesma sesión MCP durante toda a chamada. Quen ten conta co seu número no perfil inicia sesión cun código que lle chega ao correo (`phone_send_code`, `phone_verify_code`); quen non, reserva co seu nome (`book_turn` con `holder_name`), e o axente le en voz alta os detalles da reserva (`confirmation: read_out`). As instalacións só para veciños, con idade mínima e as reservas públicas piden a conta. Depende da API de [#97](https://github.com/TriboSolutions/LibreQuenda/pull/97).
- **O que di:** frases curtas e en galego; antes de reservar confirma instalación, día e hora, resume as normas de uso e o prazo de cancelación e pide que as acepte (`accept_rules`). As horas que dan as ferramentas chegan xa en hora de Galicia, e a normalización propia (`voice/lqvoice/normalize.py`) convérteas en palabras, igual que datas, números, teléfonos e abreviaturas, só para a voz: os subtítulos manteñen "15:00".
- **Latencia:** a voz sae frase a frase (e, nunha CPU, cláusula a cláusula) mentres o modelo segue escribindo. Medido nesta implementación, desde que a persoa remata a pregunta ata o primeiro son da resposta: uns **0,9 s** cunha GPU (Nós Whisper, Nós TTS, Claude Haiku; 1,3–1,8 s con Claude Sonnet) e uns **2–3,5 s** só con CPU (Nós wav2vec2, Nós TTS).

## Acendela

Con `install.sh config` (sección «Liña de voz»), ou a man en `.env`:

```sh
COMPOSE_PROFILES=voice        # nunha CPU; voice-gpu cunha GPU NVIDIA (NVIDIA Container Toolkit)
VOICE_LLM=anthropic
VOICE_LLM_API_KEY=sk-ant-…
```

e `docker compose up -d --wait`. O primeiro arranque descarga os modelos (uns 2 GB) no volume `voice-models`; o servizo está `healthy` cando xa os cargou. Precisa uns 4 GB de RAM libres na CPU, ou 4 GB de VRAM coa GPU. A imaxe `librequenda-voice` é para CPU; a de GPU constrúese con `docker build --build-arg VARIANT=cuda -t ghcr.io/tribosolutions/librequenda-voice:latest-cuda voice`.

Detrás doutro proxy, este ten que deixar pasar WebSockets en `/voice/ws` (en Nginx Proxy Manager, «Websockets Support»; o `nginx-librequenda.conf` do instalador xa o fai).

**Noutra máquina** (por exemplo, unha con GPU e o servidor principal sen ela): alí, `voice/` co seu `.env` (`VOICE_MCP_URL=https://<WEB_DOMAIN>/api/mcp`, `PUBLIC_URL=https://<WEB_DOMAIN>` e a mesma `VOICE_API_KEY`), publicado en HTTPS; no servidor principal, `VOICE_URL=https://<enderezo-da-voz>` en vez dun perfil. O proxy segue servindo `/voice/*` desde o mesmo dominio.

Para desenvolver: `cd voice && uv sync --extra cuda` (ou `--extra cpu`), `uv run python -m lqvoice` con `VOICE_MCP_URL` apuntando á API, e `pnpm dev` na web reenvía `/voice` a `VITE_VOICE_PROXY_TARGET`. Probas: `uv run pytest` e `uv run ruff check`; a proba de extremo a extremo (`uv run python e2e/smoke.py --url http://localhost:8000`) di unha pregunta cunha voz de Nós, envíaa en tempo real polo WebSocket e comproba que o axente chamou a `search_facilities` e respondeu con audio que, transcrito, nomea unha instalación.

## Provedores

| Parte | Variable | Valores (o primeiro, por defecto) | Onde funciona |
|---|---|---|---|
| Voz a texto | `VOICE_STT` | `nos-wav2vec2` (perfil `voice`), `nos-whisper` (perfil `voice-gpu`), `whisper`, `elevenlabs`, `deepgram` | Nós e `whisper`: no servidor. ElevenLabs, Deepgram: no provedor |
| Texto a voz | `VOICE_TTS`, `VOICE_TTS_VOICE` | `nos` (voces `brais`, `celtia`), `piper` (`es_ES-davefx-medium`), `elevenlabs` (un ID de voz) | Nós e Piper: no servidor. ElevenLabs: no provedor |
| Modelo de linguaxe | `VOICE_LLM`, `VOICE_LLM_MODEL` | `anthropic` (`claude-haiku-5-5`), `openai-compatible` (Ollama, vLLM… en `VOICE_LLM_BASE_URL`), `openai` | Anthropic, OpenAI: no provedor. `openai-compatible`: onde estea ese servidor |

- **Galego:** os dous modelos de Nós só entenden galego. Quen fale castelán enténdese a medias (o axente responde en galego). Para entender calquera idioma: `VOICE_STT=whisper` e `VOICE_STT_LANGUAGE=auto` (Whisper large-v3-turbo multilingüe, cunha GPU), aínda que entende peor o galego. Deepgram non ten galego (usa o seu modo multilingüe); ElevenLabs si.
- **Celtia** está restrinxida en Hugging Face: acepta as súas condicións coa túa conta e pon un `HF_TOKEN` en `.env`. Brais descárgase sen conta.
- **Claude:** `claude-haiku-5-5` é o máis rápido (primeiro texto en ~0,6 s) e chega para buscar e reservar; `claude-sonnet-5-5` resume mellor pero tarda máis. O pensamento ampliado vai apagado: nunha conversa por voz cada décima conta. `VOICE_LLM_BASE_URL` serve tamén para unha pasarela compatible coa API de Anthropic.
- **Outros axustes:** `VOICE_MAX_CALL_SECONDS` (600: a chamada despídese e colga ao chegar), `VOICE_MAX_CALLS` (4 á vez; despois, «a liña está ocupada»), `VOICE_TURN_SILENCE` (0,6 s de silencio para dar a quenda por rematada), `VOICE_DEVICE` (`auto`, `cuda`, `cpu`), `VOICE_LOG_LEVEL`. Mantén `VOICE_MAX_CALL_SECONDS` por debaixo da vida dos tokens de acceso (`ZITADEL_ACCESS_TOKEN_MINUTES`): a web renova o token antes de cada chamada, pero non durante ela.

### Totalmente autoaloxada

Sen ningún terceiro: `VOICE_STT=nos-whisper` (ou `nos-wav2vec2`), `VOICE_TTS=nos`, e un modelo de pesos abertos con chamadas a ferramentas no teu servidor, por exemplo con [Ollama](https://ollama.com):

```sh
VOICE_LLM=openai-compatible
VOICE_LLM_BASE_URL=http://ollama:11434/v1   # un servizo máis no mesmo compose, ou outra máquina da rede
VOICE_LLM_MODEL=qwen3:8b
```

Un modelo pequeno responde máis devagar nunha CPU e fala peor galego ca un grande: cunha GPU, un de 8 a 14 mil millóns de parámetros con chamadas a ferramentas (Qwen3, Llama 3.x, Mistral) é un bo punto de partida. O panel da web di sempre a quen se envía que (`/voice/config.json`, `thirdParties`).

## Privacidade

- **Nada se garda:** nin o audio nin a transcrición. Viven na memoria do proceso durante a chamada e desaparecen con ela. Os rexistros (JSON, en `docker compose logs voice`) din cando empeza e remata unha chamada, o seu tipo, a súa duración e a latencia de cada quenda, nunca o que se dixo nin o número que chama (o rexistro de depuración, `VOICE_LOG_LEVEL=DEBUG`, si mostra textos: só para probas).
- **Que sae do servidor, segundo os provedores:**

| Configuración | Voz da persoa | Texto da conversa | Datos das ferramentas |
|---|---|---|---|
| Nós + modelo propio (`openai-compatible` na túa rede) | non sae | non sae | non saen |
| Nós + Claude (por defecto) | non sae | Anthropic (EUA): o que se transcribe, o que responde o axente e o que devolven as ferramentas (instalacións, quendas, e coa sesión iniciada as reservas da persoa) | Anthropic, como parte da conversa |
| ElevenLabs ou Deepgram para a voz a texto | ese provedor | | |
| ElevenLabs para o texto a voz | | ElevenLabs: o que di o axente | |

- **Base legal:** cun provedor externo, este trata datos persoais por conta do concello: fai falta un contrato de encargado do tratamento (art. 28 RGPD) e, fóra do Espazo Económico Europeo, unha garantía para a transferencia internacional (cláusulas contractuais tipo ou o Marco de Privacidade de Datos UE-EUA). Anthropic, OpenAI, ElevenLabs e Deepgram ofrecen o seu acordo de tratamento de datos (DPA). Sen ese contrato, usa a configuración totalmente autoaloxada. A política de privacidade da instalación debe nomear os provedores que uses.
- **Contas:** a liña de voz nunca ve contrasinais. Na web usa o token da persoa só durante a chamada; por teléfono, a API acepta o número que chama só coa clave `VOICE_API_KEY`, e para usar unha conta pide o código enviado ao seu correo.

## Teléfono

**Twilio** (opcional; configurado, non despregado por defecto):

1. En `.env`: `VOICE_TWILIO_ACCOUNT_SID` e `VOICE_TWILIO_AUTH_TOKEN` (a clave `VOICE_API_KEY` que comparten a API e a liña de voz xéraa o instalador).
2. En Twilio, no número: *A call comes in* → *Webhook*, `https://<your domain>/voice/twilio`, `HTTP POST`.

A liña comproba a sinatura de Twilio en cada chamada, responde con TwiML que conecta o audio a `wss://<your domain>/voice/twilio/ws` cun tícket dun só uso, e colga ela mesma ao rematar. O audio telefónico é de 8 kHz: entende algo peor ca na web.

**SIP ou Asterisk:** un concello coa súa propia centraliña pode enviar as chamadas a un número de Twilio, ou a un *Elastic SIP Trunk* de Twilio apuntando ao mesmo webhook, sen tocar este servizo. Conectar Asterisk directamente (AudioSocket ou o WebSocket de medios de Asterisk 22) precisa un serializador de Pipecat para ese protocolo, igual que `TwilioFrameSerializer`: o resto (`lqvoice/bot.py`) non cambia.

## Modelos e licenzas

| Compoñente | Licenza | Notas |
|---|---|---|
| [Pipecat](https://github.com/pipecat-ai/pipecat) | BSD-2-Clause | o fluxo da chamada, os transportes e os adaptadores dos provedores |
| [proxectonos/whisper-large-v3-turbo-gl-v1.0](https://huggingface.co/proxectonos/whisper-large-v3-turbo-gl-v1.0) | Apache-2.0 | Whisper large-v3-turbo (OpenAI, MIT) afinado para galego; convértese unha vez a CTranslate2 |
| [proxectonos/Nos_ASR-wav2vec2-xls-r-300m-gl](https://huggingface.co/proxectonos/Nos_ASR-wav2vec2-xls-r-300m-gl) | Apache-2.0 | sobre XLS-R 300M (Meta, Apache-2.0) |
| [proxectonos/Nos_TTS-brais-vits-graphemes](https://huggingface.co/proxectonos/Nos_TTS-brais-vits-graphemes), [celtia](https://huggingface.co/proxectonos/Nos_TTS-celtia-vits-graphemes) | Apache-2.0 | coas condicións de uso de Nós en Hugging Face (recoñecer a autoría, non usalos para desinformar) |
| [coqui-tts](https://github.com/idiap/coqui-ai-TTS) (o fork mantido de Coqui TTS) | MPL-2.0 | executa as voces VITS |
| [faster-whisper](https://github.com/SYSTRAN/faster-whisper), [CTranslate2](https://github.com/OpenNMT/CTranslate2) | MIT | |
| [Silero VAD](https://github.com/snakers4/silero-vad) | MIT | vén con Pipecat |
| [Piper](https://github.com/OHF-Voice/piper1-gpl) | GPL-3.0 | opcional; cada voz ten a súa licenza na súa ficha |
| `@pipecat-ai/client-js`, `@pipecat-ai/websocket-transport` | BSD-2-Clause | o cliente da web |

Os pesos non van na imaxe nin no repositorio: descárganse de Hugging Face no primeiro arranque.
