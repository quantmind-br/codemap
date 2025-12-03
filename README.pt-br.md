# codemap 🗺️

> **codemap — um cérebro de projeto para sua IA.**
> Dê aos LLMs contexto arquitetônico instantâneo sem queimar tokens.

![Licença](https://img.shields.io/badge/license-MIT-blue.svg)
![Go](https://img.shields.io/badge/go-1.21+-00ADD8.svg)

![captura de tela do codemap](assets/codemap.png)

## Índice

- [Por que o codemap existe](#por-que-o-codemap-existe)
- [Recursos](#recursos)
- [Como Funciona](#️-como-funciona)
- [Performance](#-performance)
- [Instalação](#instalação)
- [Uso](#uso)
- [Modo Diff](#modo-diff)
- [Modo Fluxo de Dependências](#modo-fluxo-de-dependências)
- [Modo Skyline](#modo-skyline)
- [Linguagens Suportadas](#linguagens-suportadas)
- [Integrações com Claude](#integrações-com-claude)
- [Roadmap](#roadmap)
- [Contribuindo](#contribuindo)
- [Licença](#licença)

## Por que o codemap existe

LLMs modernos são poderosos, mas cegos. Eles conseguem escrever código — mas só depois de você pedir para eles queimarem tokens procurando ou manualmente explicando toda a estrutura do seu projeto.

Isso significa:
*   🔥 **Queimando milhares de tokens**
*   🔁 **Repetindo contexto**
*   📋 **Colando árvores de diretórios**
*   ❓ **Respondendo "onde X está definido?"**

**O codemap resolve isso.**

Um comando → um "mapa cerebral" compacto e estruturado da sua base de código que os LLMs podem entender instantaneamente.

## Recursos

- 🧠 **Saída de Mapa Cerebral**: Visualiza a estrutura da sua base de código em um único bloco colável.
- 📉 **Econômico em Tokens**: Agrupa arquivos e simplifica nomes para economizar espaço vertical.
- ⭐️ **Destaque Inteligente**: Sinaliza automaticamente os 5 maiores arquivos de código fonte.
- 📂 **Achatamento Inteligente**: Mescla diretórios intermediários vazios (ex: `src/main/java`).
- 🎨 **Contexto Rico**: Codificado por cores por linguagem para fácil visualização.
- 🚫 **Redução de Ruído**: Ignora automaticamente `.git`, `node_modules` e assets (imagens, binários).

## ⚙️ Como Funciona

**codemap** é um único binário Go — rápido e sem dependências:
1.  **Scanner**: Atravessa instantaneamente seu diretório, respeitando `.gitignore` e ignorando arquivos indesejados.
2.  **Analisador**: Usa gramáticas tree-sitter para analisar imports/funcções em 16 linguagens.
3.  **Renderizador**: Produz um "mapa cerebral" limpo e denso que é legível por humanos e otimizado para LLMs.

## ⚡ Performance

**codemap** roda instantaneamente mesmo em repositórios grandes (centenas ou milhares de arquivos). Isso o torna ideal para workflows com LLMs — sem lag, sem dança de múltiplas ferramentas.

## Instalação

### Homebrew (macOS/Linux)

```bash
brew tap JordanCoin/tap
brew install codemap
```

### Scoop (Windows)

```powershell
scoop bucket add codemap https://github.com/JordanCoin/scoop-codemap
scoop install codemap
```

### Download do Binário

Binários pré-compilados com suporte completo para `--deps` estão disponíveis para todas as plataformas na [página de Releases](https://github.com/JordanCoin/codemap/releases):

- **macOS**: `codemap-darwin-amd64.tar.gz` (Intel) ou `codemap-darwin-arm64.tar.gz` (Apple Silicon)
- **Linux**: `codemap-linux-amd64.tar.gz` ou `codemap-linux-arm64.tar.gz`
- **Windows**: `codemap-windows-amd64.zip`

```bash
# Exemplo: download e instalação no Linux/macOS
curl -L https://github.com/JordanCoin/codemap/releases/latest/download/codemap-linux-amd64.tar.gz | tar xz
sudo mv codemap-linux-amd64/codemap /usr/local/bin/
sudo mv codemap-linux-amd64/grammars /usr/local/lib/codemap/
```

```powershell
# Exemplo: Windows (PowerShell)
Invoke-WebRequest -Uri "https://github.com/JordanCoin/codemap/releases/latest/download/codemap-windows-amd64.zip" -OutFile codemap.zip
Expand-Archive codemap.zip -DestinationPath C:\codemap
# Adicione C:\codemap\codemap-windows-amd64 ao seu PATH
```

Cada release inclui o binário, gramáticas tree-sitter e arquivos de query para suporte completo ao `--deps`.

### A partir do código fonte

```bash
git clone https://github.com/JordanCoin/codemap.git
cd codemap
go build -o codemap .
```

## Uso

Execute `codemap` em qualquer diretório:

```bash
codemap
```

Ou especifique um caminho:

```bash
codemap /caminho/para/meu/projeto
```

### Exemplo de Uso com IA

**O Caso de Uso Matador:**

1.  Execute o codemap e copie a saída:
    ```bash
    codemap . | pbcopy
    ```

2.  Ou simplesmente diga ao Claude, Codex, ou Cursor:
    > "Use codemap para entender a estrutura do meu projeto."

## Modo Diff

Veja o que você está trabalhando com `--diff`:

```bash
codemap --diff
```

```
╭─────────────────────────── meuprojeto ──────────────────────────╮
│ Alterados: 4 arquivos | +156 -23 linhas vs main                │
│ Principais Extensões: .go (3), .tsx (1)                        │
╰────────────────────────────────────────────────────────────────╯
meuprojeto
├── api/
│   └── (novo) auth.go         ✎ handlers.go (+45 -12)
├── web/
│   └── ✎ Dashboard.tsx (+82 -8)
└── ✎ main.go (+29 -3)

⚠ handlers.go é usado por 3 outros arquivos
⚠ api é usado por 2 outros arquivos
```

**O que mostra:**
- 📊 **Resumo de alterações**: Total de arquivos e linhas alteradas vs branch main
- ✨ **Novo vs modificado**: `(novo)` para arquivos não rastreados, `✎` para modificados
- 📈 **Contagem de linhas**: `(+45 -12)` mostra adições e deleções por arquivo
- ⚠️ **Análise de impacto**: Quais arquivos alterados são importados por outros (usa tree-sitter)

Compare com uma branch diferente:
```bash
codemap --diff --ref develop
```

## Modo Fluxo de Dependências

Veja como seu código se conecta com `--deps`:

```bash
codemap --deps /caminho/para/projeto
```

```
╭──────────────────────────────────────────────────────────────╮
│                   MyApp - Fluxo de Dependências              │
├──────────────────────────────────────────────────────────────┤
│ Go: chi, zap, testify                                        │
│ Py: fastapi, pydantic, httpx                                 │
╰──────────────────────────────────────────────────────────────╯

Backend ════════════════════════════════════════════════════
  server ───▶ validate ───▶ rules, config
  api ───▶ handlers, middleware

Frontend ═══════════════════════════════════════════════════
  App ──┬──▶ Dashboard
        ├──▶ Settings
        └──▶ api

HUBS: config (12←), api (8←), utils (5←)
45 arquivos · 312 funções · 89 deps
```

**O que mostra:**
- 📦 **Dependências externas** agrupadas por linguagem (de go.mod, requirements.txt, package.json, etc.)
- 🔗 **Cadeias de dependência internas** mostrando como os arquivos importam uns aos outros
- 🎯 **Arquivos Hub** — os arquivos mais importados da sua base de código

## Modo Skyline

Quer algo mais visual? Execute `codemap --skyline` para uma visualização em formato de paisagem urbana da sua base de código:

```bash
codemap --skyline --animate
```

![skyline do codemap](assets/skyline-animated.gif)

Cada prédio representa uma linguagem no seu projeto — prédios mais altos significam mais código. Adicione `--animate` para prédios subindo, estrelas piscando e estrelas cadentes.

## Linguagens Suportadas

O codemap suporta **16 linguagens** para análise de dependências:

| Linguagem | Extensões | Detecção de Import |
|-----------|-----------|-------------------|
| Go | .go | declarações import |
| Python | .py | import, from...import |
| JavaScript | .js, .jsx, .mjs | import, require |
| TypeScript | .ts, .tsx | import, require |
| Rust | .rs | use, mod |
| Ruby | .rb | require, require_relative |
| C | .c, .h | #include |
| C++ | .cpp, .hpp, .cc | #include |
| Java | .java | import |
| Swift | .swift | import |
| Kotlin | .kt, .kts | import |
| C# | .cs | using |
| PHP | .php | use, require, include |
| Dart | .dart | import |
| R | .r, .R | library, require, source |
| Bash | .sh, .bash | source, . |

## Integrações com Claude

O codemap oferece três formas de integração com Claude:

### CLAUDE.md (Recomendado)

Adicione o `CLAUDE.md` incluído à raiz do seu projeto. O Claude Code lê automaticamente e sabe quando executar o codemap:

```bash
cp /caminho/para/codemap/CLAUDE.md seu-projeto/
```

Isso ensina o Claude a:
- Executar `codemap .` antes de iniciar tarefas
- Executar `codemap --deps` ao refatorar
- Executar `codemap --diff` ao revisar alterações

### Skill do Claude Code

Para invocação automática, instale a skill do codemap:

```bash
# Copie para seu projeto
cp -r /caminho/para/codemap/.claude/skills/codemap seu-projeto/.claude/skills/

# Ou instale globalmente
cp -r /caminho/para/codemap/.claude/skills/codemap ~/.claude/skills/
```

Skills são invocadas pelo modelo — Claude decide automaticamente quando usar o codemap baseado nas suas perguntas, sem necessidade de comandos explícitos.

### Servidor MCP

Para a integração mais profunda, execute o codemap como um servidor MCP:

```bash
# Compile o servidor MCP
make build-mcp

# Adicione ao Claude Code
claude mcp add --transport stdio codemap -- /caminho/para/codemap-mcp
```

Ou adicione ao `.mcp.json` do seu projeto:

```json
{
  "mcpServers": {
    "codemap": {
      "command": "/caminho/para/codemap-mcp",
      "args": []
    }
  }
}
```

**Claude Desktop:**

> ⚠️ Claude Desktop não pode ver seus arquivos locais por padrão. Este servidor MCP roda na sua máquina e dá ao Claude essa capacidade.

Adicione ao `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "codemap": {
      "command": "/caminho/para/codemap-mcp"
    }
  }
}
```

**Ferramentas MCP:**
| Ferramenta | Descrição |
|------------|-----------|
| `status` | Verifica conexão MCP e acesso ao sistema de arquivos local |
| `list_projects` | Descobre projetos em um diretório pai (com filtro opcional) |
| `get_structure` | Visualização em árvore do projeto com tamanhos de arquivo e detecção de linguagem |
| `get_dependencies` | Fluxo de dependências com imports, funções e arquivos hub |
| `get_diff` | Arquivos alterados com contagem de linhas e análise de impacto |
| `find_file` | Encontra arquivos por padrão de nome |
| `get_importers` | Encontra todos os arquivos que importam um arquivo específico |

## Roadmap

- [x] **Modo Diff** (`codemap --diff`) — mostra arquivos alterados com análise de impacto
- [x] **Modo Skyline** (`codemap --skyline`) — visualização de paisagem urbana ASCII
- [x] **Fluxo de Dependências** (`codemap --deps`) — análise de função/import com suporte para 16 linguagens
- [x] **Skill Claude Code** — invocação automática baseada em perguntas do usuário
- [x] **Servidor MCP** — integração profunda com 7 ferramentas para análise de base de código

## Contribuindo

Adoramos contribuições!
1.  Faça o fork do repositório.
2.  Crie uma branch (`git checkout -b feature/minha-feature`).
3.  Commit suas alterações.
4.  Push e abra um Pull Request.

## Licença

MIT