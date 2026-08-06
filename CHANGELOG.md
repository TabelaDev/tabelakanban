# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.2.0] - 2026-08-06

### Added

- Colunas agora redimensionam pra preencher a largura da tela, em vez de
  ficarem num tamanho fixo estreito.
- Sidebar de boards como coluna à esquerda, visível por padrão e colapsável
  com `ctrl+e` — substitui o modal de seleção.
- Preview do markdown do card numa coluna lateral com `o`.
- Página de log de atividades com `g`: registra card/board/colunas de origem
  e destino de cada ação, renderizada em página cheia.
- Linha reservada entre as colunas e o footer só para as notificações de
  ações, que somem em ~2s.
- `ctrl+r` agora dá retorno ("recarregado — N boards").
- Renomear coluna (`R`), apagar coluna (`D`, com confirmação) e reordenar
  colunas (`ctrl+h`/`ctrl+l`), via arquivo `.order` do board.
- Due date nos cards: `t` define/limpa (formato `DD/MM`), guardado no
  front-matter (`due: YYYY-MM-DD`) e exibido no rodapé do card (vermelho
  quando vencido).
- Prompts de input viraram formulários estilo djobs (`huh`): campos com
  label, placeholder e navegação — criar card já pergunta o due.

### Changed

- Navegação hjkl: `h`/`l` navegam colunas (e a sidebar), `j`/`k` cards,
  `H`/`L` movem card.
- Header/footer com background do accent e padding interno, colados nas
  bordas do terminal (sem margem externa).
- Cards renderizados em blocos com separador; o preview embaixo do título
  pula o `# H1` do arquivo (que repetia o título); a seleção agora é um
  retângulo accent completo (pad antes de aplicar o style).

### Fixed

- Layout com o input de novo card/coluna aberto (e em board sem colunas):
  o rodapé ficava subindo; agora o estado vazio preenche a altura toda e o
  rodapé fica ancorado embaixo.
- Notificação não desrenderiza mais a borda das colunas.
- Com o preview aberto, a sidebar e o header não encolhem mais nem saem da
  tela (o painel de preview tinha 4 linhas a mais que as colunas).
- Modal de input não estourava mais a largura do terminal (prompt duplicado
  removido).
