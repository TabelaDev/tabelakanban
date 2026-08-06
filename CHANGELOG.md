# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Colunas agora redimensionam pra preencher a largura da tela, em vez de
  ficarem num tamanho fixo estreito.
- Sidebar de boards como coluna à esquerda, visível por padrão e colapsável
  com `ctrl+e` — substitui a faixa de cima e o modal de seleção.
- Preview do markdown do card numa coluna lateral com `o`.
- Página de log de atividades com `g`: registra card/board/colunas de origem
  e destino de cada ação, renderizada em página cheia.
- Linha reservada entre as colunas e o footer só para as notificações de
  ações, que somem em ~2s.
- `ctrl+r` agora dá retorno ("recarregado — N boards").
- Renomear coluna (`R`), apagar coluna (`D`, com confirmação) e reordenar
  colunas (`ctrl+shift+h/l`), via arquivo `.order` do board.
- Suporte ao protocolo de teclado do kitty (flag de disambiguation), o que
  faz `ctrl+shift+h/l` funcionar de verdade em terminais compatíveis.

### Changed

- Navegação hjkl: `h`/`l` navegam colunas (e a sidebar), `j`/`k` cards,
  `H`/`L` movem card.
- Prompt de novo card/coluna/board virou um modal centralizado, em vez de
  uma linha que empurrava o layout.
- Header passou a usar o background do accent (vem da tabelatuiui) e o frame
  inteiro ganhou margem das bordas do terminal; padding interno do
  header/footer foi reduzido (também na tabelatuiui).
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
- Input não funcionava com o reader de CSI-u: o wrapper agora implementa
  `term.File` (`Fd`), então o bubbletea ainda ativa o raw mode (sem isso o
  terminal ecoava as teclas e nada respondia).
