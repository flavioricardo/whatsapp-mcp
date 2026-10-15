# STATE — whatsapp-mcp

> **Convenção deste arquivo:** só fato vivo — o que é verdade hoje e o que está
> aberto. Histórico de como chegamos aqui vive no `git log`. Ao fechar uma
> pendência, apague-a daqui (o commit registra); se ela deixou algo ainda por
> fazer, promova essa linha para Pendências. Nunca seção "Resolvidas"/"Feito"
> (o CI barra). Limite: ~150 linhas.

## Status

- **Fase:** em uso pessoal diário — MCP `whatsapp` conectado ao Claude Code local.
- **No ar:** só local, sem deploy: bridge Go (`whatsapp-bridge/`, whatsmeow +
  SQLite) e servidor MCP Python (`whatsapp-mcp-server/`).

## Decisões

- Fork de `lharries/whatsapp-mcp`: a bridge do upstream é rejeitada pelo
  WhatsApp; este fork reconecta e soma transcrição local (faster-whisper),
  legendas de mídia e listas/etiquetas do Business (ver README).

## Pendências

<!-- Formato: - [ ] **Ação exata** → link direto | trava: … | [humano] se só o Flávio fecha -->
