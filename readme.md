# Iran Domains Telegram Bot

A bot that collects Iranian hosted applications, or websites domain names from user submissions.
It's accessible on Telegram with username: [`@iran_domains_bot`](http://t.me/iran_domains_bot).

## Usage

1. Download the latest executable and run it!

    ```sh
    curl -SfLO https://github.com/z4x7k/iran-domains-tg-bot/releases/latest/download/bot && chmod +x ./bot
    ```

    Verify it's working using `./bot --help` command.

2. Download `.env` template file:

    ```sh
    curl -SfLo .ir-domains-bot.env https://raw.githubusercontent.com/z4x7k/iran-domains-tg-bot/main/.env.template
    ```

3. Set proper values in `.env`
4. Run it!

    ```sh
    ./bot run --db ir-domains.db --env .env
    ```

## SystemD Service Unit

Write the content below in a service unit file, e.g., `~/.config/systemd/user/ir-domains-bot.service`

```service
[Unit]
Description=Iran Domains Telegram Bot
After=network.target
Wants=network-online.target

[Service]
Restart=on-failure
Type=simple
ExecStart=path_to_bot_executable run --db path_to_db_file.db --env path_to_dotenv
RestartSec=10s
TimeoutStopSec=20s
KillSignal=SIGINT
FinalKillSignal=SIGKILL

[Install]
WantedBy=multi-user.target
```

Then execute the following commands to activate, start, and enable start-on-boot the service:

```sh
systemctl --user daemon-reload
systemctl --user start ir-domains-bot.service
systemctl --user enable ir-domains-bot.service
```

**Note**: in order for start-on-boot to work for user service unit to work, run the following command if you haven't already:

```sh
sudo loginctl enable-linger $USER
```
