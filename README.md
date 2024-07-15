# GBBS (Go Bulletin Board System)

GBBS is a modern implementation of a classic Bulletin Board System (BBS) written in Go. It provides a nostalgic interface with modern backend technologies, supporting Telnet, SSH, and Web access.

## Features

- Multi-protocol support: Telnet, SSH, and Web
- User authentication and registration
- Message board functionality
- ANSI color support for Telnet and SSH clients
- Customizable welcome screen
- SQLite database for user management
- Concurrent connections handling

## Getting Started

### Prerequisites

- Go 1.16 or later
- SQLite

### Installation

1. Clone the repository:
   ```
   git clone https://github.com/strangeprogram/gbbs.git
   cd gbbs
   ```

2. Install dependencies:
   ```
   go mod tidy
   ```

3. Create a `config.json` file in the project root:
   ```json
   {
       "telnet_port": 2323,
       "ssh_port": 2222,
       "web_port": 8080,
       "guestbook_path": "guestbook.txt",
       "web_root": "web",
       "welcome_screen_path": "welcome.ans"
   }
   ```

4. Create a `welcome.ans` file with your desired ANSI art welcome screen.

### Running the BBS

To run the BBS in debug mode:

```
go run cmd/gbbs/main.go --debug
```

To compile and run:

```
go build -o gbbs cmd/gbbs/main.go
./gbbs
```

## Connecting to the BBS

- Telnet: `telnet localhost 2323`
- SSH: `ssh localhost -p 2222`
- Web: Open a browser and navigate to `http://localhost:8080`

## Version History

- v0.1: Initial implementation with basic Telnet support
- v0.2: Added SSH support
- v0.3: Implemented web interface
- v0.4: Added user authentication and registration
- v0.5: Introduced message board functionality
- v0.6: Improved ANSI color support and welcome screen customization
- v0.7: Fixed SSH input handling issues

## TODO

- [ ] Implement IRC link integration
- [ ] Add file transfer capabilities
- [ ] Create a more robust web interface
- [ ] Implement user roles and permissions
- [ ] Add support for multiple message boards/forums
- [ ] Implement private messaging between users
- [ ] Create a plugin system for easy feature extensions
- [ ] Add support for external authentication methods (e.g., OAuth)
- [ ] Implement a basic game or interactive feature
- [ ] Create a telnet/SSH client specifically designed for this BBS

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the GNU License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- MysticBBS | Oblivion/v2
- BBS
- archive the dream