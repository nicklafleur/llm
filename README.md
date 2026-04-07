# Local LLM tooling

Configuration and helpers for running [LLaMA.cpp](https://github.com/ggml-org/llama.cpp) models locally (Continue or any OpenAI-compatible client).

## Quick start

1. Build [llama.cpp](https://github.com/ggml-org/llama.cpp) and put `llama-server`, `llama-cli`, and `llama-bench` on your `PATH`.
2. From the repo root, build the CLI:

   ```bash
   go build -o llm .
   ```

3. **Multi-model API server** (loads every `[section]` from `srv.ini` via `--models-preset`):

   ```bash
   ./llm srv
   ./llm srv turbo.ini
   ```

4. **Single-model run** (reads one section from `srv.ini`):

   ```bash
   ./llm run qwen30b
   ./llm run turbo.ini qwen30b
   ./llm run qwen30b -c          # llama-cli
   ./llm run qwen30b -b          # llama-bench
   ./llm run qwen30b -q 6        # override GGUF quant (see below)
   ```

## CLI reference (`llm`)

| Command | Purpose |
|--------|---------|
| `llm srv [ini]` | `llama-server --models-preset <srv.ini>` |
| `llm run [ini] <model> [options] [--] [args...]` | `llama-server`, `llama-cli`, or `llama-bench` for one model |

Global flag:

- `--config <path>` — path to `srv.ini`. If omitted: `llm_INI` env var, then `./srv.ini`, then `bin/srv.ini`, then paths next to the executable.
- Both `srv` and `run` also accept an optional positional `.ini` path, which overrides `--config`.

### `run` options (after the model name)

- `-c` — interactive CLI (`llama-cli`).
- `-b` — benchmark (`llama-bench`).
- `-q <N>` — override the quant suffix in `hf-repo` for models that define presets in code: **qwen7b** `4`, `8`; **qwen30b** `2`, `4`, `6`; **qwen80b** `2`, `3`, `3s`, `4`, `4s`. Other sections rely on `hf-repo` in `srv.ini` only.

In `llm run`, the model name `default` selects the **qwen80b** section (there is no `[default]` block in `srv.ini`).

## `srv.ini`

INI sections define each model. Only keys that are present are turned into flags; missing keys use the llama binary defaults. Typical keys:

| Key | Used for |
|-----|----------|
| `hf-repo` | Hugging Face repo id and default quant, `repo:QUANT` (required) |
| `ctx-size` | Context length (server/cli when set) |
| `flash-attn` | Passed to `llama-bench` when set |
| `ngl`, `cache-type-k`, `cache-type-v`, `n-cpu-moe` | GPU/cache/MoE-related flags when set |
| `mmap` | If `disabled` / `off` / `false` / `0` / `no`, adds `--no-mmap` |

Sections in this repo:

- **qwen30b** — `unsloth/Qwen3-Coder-30B-A3B-Instruct-GGUF` (default quant in ini: `UD-Q6_K_XL`).
- **qwen80b** — `unsloth/Qwen3-Coder-Next-GGUF` (`IQ4_NL` in ini).
- **qwen7b** — `Qwen/Qwen2.5-Coder-7B-Instruct-GGUF` (`Q8_0` in ini).
- **nomic-embed** — embeddings model (`F32` in ini).
- **qwen-rerank** — reranker GGUF.
- **fast-apply** — small apply model.

## IDE integration (Continue)

Point the Continue extension (or any client) at your local OpenAI-compatible base URL (for example `http://localhost:8080` or whatever host/port `llama-server` uses). Model IDs and routing depend on how you start the server (`preset` vs single `run`) and your client config.

## Requirements

- Go 1.22+ (to build `llm`).
- LLaMA.cpp binaries on `PATH`: at least `llama-server`; also `llama-cli` and `llama-bench` if you use `llm run -c` / `-b`.
