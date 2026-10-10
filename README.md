# WhatsApp MCP Server

This is a Model Context Protocol (MCP) server for WhatsApp.

With this you can search and read your personal Whatsapp messages (including images, videos, documents, and audio messages), search your contacts and send messages to either individuals or groups. You can also send media files including images, videos, documents, and audio messages.

It connects to your **personal WhatsApp account** directly via the Whatsapp web multidevice API (using the [whatsmeow](https://github.com/tulir/whatsmeow) library). All your messages are stored locally in a SQLite database and only sent to an LLM (such as Claude) when the agent accesses them through tools (which you control).

Here's an example of what you can do when it's connected to Claude.

![WhatsApp MCP](./example-use.png)

> To get updates on this and other projects I work on [enter your email here](https://docs.google.com/forms/d/1rTF9wMBTN0vPfzWuQa2BjfGKdKIpTbyeKxhPMcEzgyI/preview)

> *Caution:* as with many MCP servers, the WhatsApp MCP is subject to [the lethal trifecta](https://simonwillison.net/2025/Jun/16/the-lethal-trifecta/). This means that project injection could lead to private data exfiltration.

## About this fork

This is a fork of [lharries/whatsapp-mcp](https://github.com/lharries/whatsapp-mcp) that makes the bridge connect to WhatsApp again (upstream's version is rejected) and adds a few features.

### New features

- **Audio transcription:** the `transcribe_audio` tool turns voice and audio messages into text. It runs locally with [faster-whisper](https://github.com/SYSTRAN/faster-whisper), so it needs no API key and the audio never leaves your machine.
- **Media captions:** the text sent with an image, video or document is stored as the message content, so it shows up in `list_messages` and in searches.
- **Chat lists and Business labels:** the bridge syncs the lists you see as filters in WhatsApp's Chats tab (Favorites, your custom lists) and WhatsApp Business labels.
  - `list_chats` and `get_chat` show the lists each chat is in, and messages show `(Lists: ...)`.
  - `list_chats` and `list_messages` accept a `label` filter: the exact list name, ignoring upper/lower case (accented letters included).
  - The new `list_labels` tool lists them all with their chat counts.

### Fixes

- **The bridge connects again:** WhatsApp rejected the old whatsmeow version ("Client outdated (405)", no QR code). whatsmeow is updated, which needs Go 1.26+.
- **Chats keyed by phone number:** WhatsApp now addresses chats by LID (`@lid`). The bridge converts LIDs to phone-number JIDs, so contact search and sending by phone number work again. Chats stored under LIDs are migrated on startup when their phone number is known.
- **Media downloads work again:** `download_media` failed for every message (HTTP 403), because the signed part of the media URL was dropped.
- **Downloaded file names:** files are saved as `<message id>_<filename>`. Before, two files received in the same second got the same name, and a sender could choose a document name that wrote outside `store/`.
- **Chat names:** direct chats were sometimes named after your own number. Names now come from the contact (saved name, then profile name, then business name), else the phone number. Existing chats are fixed on startup.
- **`list_chats` / `get_chat` with `include_last_message=False`** returned nothing; they now return the chats.
- **History sync** now reads disappearing, view-once and document-with-caption messages the same way as live ones.
- The MCP server's log messages no longer go to stdout, where they could corrupt the replies sent to Claude.

### Upgrading from upstream

If you already run upstream:

1. Switch your clone to this fork:

   ```bash
   git remote set-url origin https://github.com/flavioricardo/whatsapp-mcp.git
   git pull
   ```

2. Install the new Python dependencies once, so the MCP server's first start isn't slow:

   ```bash
   cd whatsapp-mcp-server
   uv sync
   ```

3. Stop the bridge and start it again (`cd whatsapp-bridge && go run main.go`), then restart Claude Desktop / Cursor.

Start the bridge first: it creates the new tables and syncs your existing lists. Until it does, `list_chats` and `get_chat` return nothing. Captions are stored only for messages received after the upgrade, and media downloaded before the upgrade is fetched again under the new file name.

## Installation

### Prerequisites

- Go 1.26+
- Python 3.11+
- Anthropic Claude Desktop app (or Cursor)
- UV (Python package manager), install with `curl -LsSf https://astral.sh/uv/install.sh | sh`
- FFmpeg (_optional_) - Only needed for audio messages. If you want to send audio files as playable WhatsApp voice messages, they must be in `.ogg` Opus format. With FFmpeg installed, the MCP server will automatically convert non-Opus audio files. Without FFmpeg, you can still send raw audio files using the `send_file` tool.

### Steps

1. **Clone this repository**

   ```bash
   git clone https://github.com/flavioricardo/whatsapp-mcp.git
   cd whatsapp-mcp
   ```

2. **Run the WhatsApp bridge**

   Navigate to the whatsapp-bridge directory and run the Go application:

   ```bash
   cd whatsapp-bridge
   go run main.go
   ```

   The first time you run it, you will be prompted to scan a QR code. Scan the QR code with your WhatsApp mobile app to authenticate.

   After approximately 20 days, you will might need to re-authenticate.

3. **Connect to the MCP server**

   Copy the below json with the appropriate {{PATH}} values:

   ```json
   {
     "mcpServers": {
       "whatsapp": {
         "command": "{{PATH_TO_UV}}", // Run `which uv` and place the output here
         "args": [
           "--directory",
           "{{PATH_TO_SRC}}/whatsapp-mcp/whatsapp-mcp-server", // cd into the repo, run `pwd` and enter the output here + "/whatsapp-mcp-server"
           "run",
           "main.py"
         ]
       }
     }
   }
   ```

   For **Claude**, save this as `claude_desktop_config.json` in your Claude Desktop configuration directory at:

   ```
   ~/Library/Application Support/Claude/claude_desktop_config.json
   ```

   For **Cursor**, save this as `mcp.json` in your Cursor configuration directory at:

   ```
   ~/.cursor/mcp.json
   ```

4. **Restart Claude Desktop / Cursor**

   Open Claude Desktop and you should now see WhatsApp as an available integration.

   Or restart Cursor.

### Windows Compatibility

If you're running this project on Windows, be aware that `go-sqlite3` requires **CGO to be enabled** in order to compile and work properly. By default, **CGO is disabled on Windows**, so you need to explicitly enable it and have a C compiler installed.

#### Steps to get it working:

1. **Install a C compiler**  
   We recommend using [MSYS2](https://www.msys2.org/) to install a C compiler for Windows. After installing MSYS2, make sure to add the `ucrt64\bin` folder to your `PATH`.  
   → A step-by-step guide is available [here](https://code.visualstudio.com/docs/cpp/config-mingw).

2. **Enable CGO and run the app**

   ```bash
   cd whatsapp-bridge
   go env -w CGO_ENABLED=1
   go run main.go
   ```

Without this setup, you'll likely run into errors like:

> `Binary was compiled with 'CGO_ENABLED=0', go-sqlite3 requires cgo to work.`

## Architecture Overview

This application consists of two main components:

1. **Go WhatsApp Bridge** (`whatsapp-bridge/`): A Go application that connects to WhatsApp's web API, handles authentication via QR code, and stores message history in SQLite. It serves as the bridge between WhatsApp and the MCP server.

2. **Python MCP Server** (`whatsapp-mcp-server/`): A Python server implementing the Model Context Protocol (MCP), which provides standardized tools for Claude to interact with WhatsApp data and send/receive messages.

### Data Storage

- All message history is stored in a SQLite database within the `whatsapp-bridge/store/` directory
- The database maintains tables for chats and messages
- Messages are indexed for efficient searching and retrieval

## Usage

Once connected, you can interact with your WhatsApp contacts through Claude, leveraging Claude's AI capabilities in your WhatsApp conversations.

### MCP Tools

Claude can access the following tools to interact with WhatsApp:

- **search_contacts**: Search for contacts by name or phone number
- **list_messages**: Retrieve messages with optional filters and context
- **list_chats**: List available chats with metadata
- **get_chat**: Get information about a specific chat
- **get_direct_chat_by_contact**: Find a direct chat with a specific contact
- **get_contact_chats**: List all chats involving a specific contact
- **get_last_interaction**: Get the most recent message with a contact
- **get_message_context**: Retrieve context around a specific message
- **send_message**: Send a WhatsApp message to a specified phone number or group JID
- **send_file**: Send a file (image, video, raw audio, document) to a specified recipient
- **send_audio_message**: Send an audio file as a WhatsApp voice message (requires the file to be an .ogg opus file or ffmpeg must be installed)
- **download_media**: Download media from a WhatsApp message and get the local file path
- **transcribe_audio**: Transcribe an audio/voice message to text, locally with [faster-whisper](https://github.com/SYSTRAN/faster-whisper) (no API key). Set `WHISPER_MODEL` (default `small`) to trade speed for accuracy; the first call downloads the model (~500 MB for `small`) and can take a few minutes
- **list_labels**: List your WhatsApp chat lists (and WhatsApp Business labels). `list_chats` and `list_messages` show each chat's lists and accept a `label` filter

### Media Handling Features

The MCP server supports both sending and receiving various media types:

#### Media Sending

You can send various media types to your WhatsApp contacts:

- **Images, Videos, Documents**: Use the `send_file` tool to share any supported media type.
- **Voice Messages**: Use the `send_audio_message` tool to send audio files as playable WhatsApp voice messages.
  - For optimal compatibility, audio files should be in `.ogg` Opus format.
  - With FFmpeg installed, the system will automatically convert other audio formats (MP3, WAV, etc.) to the required format.
  - Without FFmpeg, you can still send raw audio files using the `send_file` tool, but they won't appear as playable voice messages.

#### Media Downloading

By default, just the metadata of the media is stored in the local database. The message will indicate that media was sent. To access this media you need to use the download_media tool which takes the `message_id` and `chat_jid` (which are shown when printing messages containing the meda), this downloads the media and then returns the file path which can be then opened or passed to another tool.

For audio and voice messages, `transcribe_audio` (same arguments) downloads the audio and returns its text. It doesn't need FFmpeg. The transcript is saved next to the audio file, so asking again with the same model is instant.

## Technical Details

1. Claude sends requests to the Python MCP server
2. The MCP server queries the Go bridge for WhatsApp data or directly to the SQLite database
3. The Go accesses the WhatsApp API and keeps the SQLite database up to date
4. Data flows back through the chain to Claude
5. When sending messages, the request flows from Claude through the MCP server to the Go bridge and to WhatsApp

## Troubleshooting

- If you encounter permission issues when running uv, you may need to add it to your PATH or use the full path to the executable.
- Make sure both the Go application and the Python server are running for the integration to work properly.

### Authentication Issues

- **QR Code Not Displaying**: If the QR code doesn't appear, try restarting the authentication script. If issues persist, check if your terminal supports displaying QR codes.
- **WhatsApp Already Logged In**: If your session is already active, the Go bridge will automatically reconnect without showing a QR code.
- **Device Limit Reached**: WhatsApp limits the number of linked devices. If you reach this limit, you'll need to remove an existing device from WhatsApp on your phone (Settings > Linked Devices).
- **No Messages Loading**: After initial authentication, it can take several minutes for your message history to load, especially if you have many chats.
- **WhatsApp Out of Sync**: If your WhatsApp messages get out of sync with the bridge, delete both database files (`whatsapp-bridge/store/messages.db` and `whatsapp-bridge/store/whatsapp.db`) and restart the bridge to re-authenticate.

For additional Claude Desktop integration troubleshooting, see the [MCP documentation](https://modelcontextprotocol.io/quickstart/server#claude-for-desktop-integration-issues). The documentation includes helpful tips for checking logs and resolving common issues.
