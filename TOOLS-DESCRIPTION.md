A ferramenta do servidor MCP (`codemap-mcp`) expõe as funcionalidades de análise da base de código, do grafo de conhecimento e do LLM para agentes de codificação.

As ferramentas disponíveis são:

---

## 🛠️ Tools do Servidor MCP

As ferramentas são implementadas nos *handlers* do arquivo `mcp/main.go` e utilizam as capacidades dos pacotes `scanner`, `graph` e `analyze`.

### 1. `get_structure` (Visão Geral da Estrutura)

| Campo | Tipo | Descrição |
| :--- | :--- | :--- |
| **`path`** | `string` | Caminho para a raiz do projeto. |

* **O que faz:** Retorna uma **visualização em árvore** da estrutura do projeto. Inclui detecção de linguagem, tamanho dos arquivos e estimativa de *tokens* por arquivo.
* **Exemplo de Utilidade:** Um agente a utiliza como **primeiro passo** para ter uma visão de alto nível do projeto, identificar os diretórios principais e estimar o tamanho do contexto de cada arquivo (`[!]` para arquivos grandes > 8k tokens).

---

### 2. `get_dependencies` (Análise de Dependências)

| Campo | Tipo | Descrição |
| :--- | :--- | :--- |
| **`path`** | `string` | Caminho para a raiz do projeto. |
| **`detail`** | `int` | Nível de detalhe: `0` (nomes), `1` (assinaturas), `2` (completo). |
| **`mode`** | `string` | Modo de saída: `deps` (fluxo de dependência, padrão) ou `api` (superfície de API pública). |

* **O que faz:** Realiza uma **análise profunda de imports, funções e tipos** utilizando Tree-sitter. Retorna o fluxo de dependência interna e lista dependências externas por linguagem. O modo `api` fornece um resumo compacto de funções e tipos exportados.
* **Exemplo de Utilidade:**
    * Um agente quer saber a **estrutura pública** de um módulo específico (usando `mode="api"`) antes de chamá-lo.
    * Um agente está refatorando um arquivo e precisa ver quais *outros* arquivos ele importa e quais dependências externas ele introduz.

---

### 3. `get_symbol` (Busca por Símbolo Precisa)

| Campo | Tipo | Descrição |
| :--- | :--- | :--- |
| **`path`** | `string` | Caminho para a raiz do projeto. |
| **`name`** | `string` | Nome do símbolo para buscar (busca de substring). |
| **`kind`** | `string` | Filtro por tipo: `function`, `type` ou `all` (padrão). |
| **`file`** | `string` | Filtro por caminho de arquivo (opcional). |

* **O que faz:** Procura por **funções, métodos e tipos** por nome na base de código. Retorna uma lista de correspondências com a **localização exata** (`caminho:linha`) e a **assinatura** completa (se `detail >= 1` for usado internamente).
* **Exemplo de Utilidade:** Um agente recebe um erro sobre a função `processRequest` e precisa encontrar todas as definições dessa função rapidamente, sabendo a linha exata e a assinatura para contextualizar.

---

### 4. `semantic_search` (Busca Híbrida)

| Campo | Tipo | Descrição |
| :--- | :--- | :--- |
| **`path`** | `string` | Caminho para a raiz do projeto. |
| **`query`** | `string` | Consulta em linguagem natural (ex: "função que faz parse da config"). |
| **`limit`** | `int` | Número máximo de resultados (padrão: `10`). |
| **`expand`** | `bool` | Incluir *callers* e *callees* no resultado. |

* **O que faz:** Combina **busca vetorial** (semântica) e **busca estrutural** (por nome no grafo) usando *Reciprocal Rank Fusion*. Retorna os símbolos mais relevantes para uma consulta em linguagem natural com pontuações.
* **Exemplo de Utilidade:** Um agente recebe uma tarefa sobre "como é feito o gerenciamento de tokens" e usa esta ferramenta com `query="token management implementation"` para encontrar as funções e tipos relevantes, mesmo que o termo não esteja no nome do arquivo ou da função.

---

### 5. `explain_symbol` (Explicação com LLM)

| Campo | Tipo | Descrição |
| :--- | :--- | :--- |
| **`path`** | `string` | Caminho para a raiz do projeto. |
| **`symbol`** | `string` | Nome do símbolo para explicar. |
| **`model`** | `string` | Modelo LLM a ser usado (sobrescreve a config). |
| **`no_cache`** | `bool` | Ignora o cache para esta requisição. |

* **O que faz:** Encontra o código-fonte de um símbolo (função, tipo, método) e utiliza um LLM (configurado via `/analyze`) para gerar uma **explicação concisa e estruturada** (*Propósito, Lógica Chave, Parâmetros*). Utiliza cache para evitar chamadas duplicadas.
* **Exemplo de Utilidade:** Um agente precisa entender a lógica de `handleTracePath` antes de modificá-la. A ferramenta fornece a explicação de forma mais rápida e focada do que pedir ao LLM para adivinhar a partir do nome.

---

### 6. `summarize_module` (Sumário de Módulo com LLM)

| Campo | Tipo | Descrição |
| :--- | :--- | :--- |
| **`path`** | `string` | Caminho para a raiz do projeto. |
| **`module`** | `string` | Módulo/diretório para sumarizar (relativo ao projeto). |
| **`model`** | `string` | Modelo LLM a ser usado (sobrescreve a config). |
| **`no_cache`** | `bool` | Ignora o cache para esta requisição. |

* **O que faz:** Lê todos os arquivos-fonte de um diretório ou arquivo e envia para o LLM gerar um **sumário de alto nível** sobre o **propósito do módulo, componentes principais e dependências**.
* **Exemplo de Utilidade:** Um agente precisa de uma visão geral do pacote `/analyze` para identificar onde a lógica de LLM está centralizada antes de integrar uma nova funcionalidade de *streaming*.

---

### 7. `trace_path`, `get_callers`, `get_callees` (Consultas de Grafo)

| Ferramenta | Tipo de Consulta | Exemplo de Uso |
| :--- | :--- | :--- |
| **`trace_path`** | Busca o **caminho mais curto** de chamadas entre dois símbolos (`from` → `to`) no grafo de conhecimento. | Útil para determinar como a função `main` se conecta a uma função de persistência de dados. |
| **`get_callers`** | Encontra quais funções **chamam** um símbolo específico, mostrando a cadeia de chamadas (**dependência reversa**). | Útil para análise de impacto: "Quem me chama e será afetado se eu mudar?" |
| **`get_callees`** | Encontra quais funções são **chamadas** por um símbolo, mostrando a cadeia de dependência. | Útil para entender o fluxo de execução e as dependências diretas de uma função. |

**Exemplo de Utilidade Geral:** Depois de usar `semantic_search` para encontrar um símbolo relevante, o agente usa `get_callers` para identificar o ponto de entrada da funcionalidade, ou `trace_path` para mapear o fluxo completo de uma transação.