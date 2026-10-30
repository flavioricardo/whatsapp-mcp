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

- Bridge no Windows do Flávio roda pela tarefa agendada `WhatsAppBridge`
  (vigia `whatsapp-bridge/run-bridge.ps1`, ver README "Keep the bridge running
  on Windows"). Logs e heartbeat em `whatsapp-bridge/logs/`. A ponte grava o
  heartbeat (arquivo + tabela `bridge_status` no `messages.db`) só se conectada
  e gravando no banco; 5 min sem ele, o vigia mata e relança. O Cardápio Zap lê
  `store/messages.db`.

## Pendências

<!-- Formato: - [ ] **Ação exata** → link direto | trava: … | [humano] se só o Flávio fecha -->
