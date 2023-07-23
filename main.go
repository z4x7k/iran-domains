package main

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/joho/godotenv"
	"github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

const (
	EnvKeyBotToken                   = "BOT_TOKEN"
	EnvKeyPublishChatID              = "PUBLISH_CHAT_ID"
	EnvKeySupportUserChatID          = "SUPPORT_USER_CHAT_ID"
	EnvKeyBotHTTPProxyURL            = "BOT_HTTP_PROXY_URL"
	ParseModeMarkdownV1              = models.ParseMode("Markdown")
	CLIRunCommandName                = "run"
	CLIRunCommandDomainsFileNameFlag = "domains"
	CLIRunCommandDBFileNameFlag      = "db"
)

var (
	AppVersion     = "0.0.0"
	AppCompileTime = "1991-11-22T00:11:22+00:00"
)

func main() {
	compileTime, err := time.Parse(time.RFC3339, AppCompileTime)
	if nil != err {
		panic(err)
	}

	log := zerolog.New(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) { w.Out = os.Stderr; w.TimeFormat = time.RFC3339 })).With().Timestamp().Logger().Level(zerolog.TraceLevel)

	app := &cli.App{
		Name:           "iran-domains-tg-bot",
		Version:        AppVersion,
		Compiled:       compileTime,
		Suggest:        true,
		Usage:          "Iranian Domains Telegram Bot",
		DefaultCommand: CLIRunCommandName,
		Commands: []*cli.Command{
			{
				Name:   CLIRunCommandName,
				Usage:  "Start the bot server",
				Action: buildBot(log),
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     CLIRunCommandDomainsFileNameFlag,
						Aliases:  []string{"f"},
						Usage:    "Domains file name. Defaults to domains.txt in the current working directory",
						Required: false,
					},
					&cli.StringFlag{
						Name:     CLIRunCommandDBFileNameFlag,
						Usage:    "Database file name. Defaults to domains.db in the current working directory",
						Required: false,
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("command failed")
	}
}

func buildBot(log zerolog.Logger) func(*cli.Context) error {
	return func(cliCtx *cli.Context) error {
		ctx, cancel := signal.NotifyContext(cliCtx.Context, os.Interrupt)
		defer cancel()

		if err := godotenv.Load(); nil != err {
			if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("dotenv: unexpected error while loading environment variables from .env file")
			}
			log.Warn().Msg(".env file not found")
		}

		dbFileName := cliCtx.String(CLIRunCommandDBFileNameFlag)
		if dbFileName == "" {
			dbFileName = "domains.db"
		}

		db, err := sql.Open("sqlite3", dbFileName)
		if nil != err {
			return fmt.Errorf("db: unable to open database: %v", err)
		}
		defer func() {
			log.Info().Msg("closing database connection")
			if err := db.Close(); nil != err {
				log.Error().Err(err).Msg("failed to close database connection")
			}
		}()
		if err := db.PingContext(ctx); nil != err {
			return fmt.Errorf("db: unable to ping database connection: %v", err)
		}
		sqliteLibVersion, sqliteLibVersionNumber, _ := sqlite3.Version()
		log.Info().Str("lib_version", sqliteLibVersion).Int("lib_version_number", sqliteLibVersionNumber).Msg("successfully connected to sqlite database")
		if err := execPragmas(ctx, db); nil != err {
			return fmt.Errorf("db: unable to execute database pragmas: %v", err)
		}
		log.Info().Msg("db: successfully executed database pragmas")

		publishChatID, ok := os.LookupEnv(EnvKeyPublishChatID)
		if !ok {
			log.Fatal().Str("key", EnvKeyPublishChatID).Msg("required environment variable is not set")
		}

		supportUserChatID, ok := os.LookupEnv(EnvKeySupportUserChatID)
		if !ok {
			log.Fatal().Str("key", EnvKeySupportUserChatID).Msg("required environment variable is not set")
		}

		domainsFileName := cliCtx.String(CLIRunCommandDomainsFileNameFlag)
		if domainsFileName == "" {
			domainsFileName = "domains.txt"
		}

		handler := Handler{
			log:               log,
			domainsFileName:   domainsFileName,
			publishChatID:     publishChatID,
			supportUserChatID: supportUserChatID,
			db:                db,
		}

		httpTransport := http.Transport{IdleConnTimeout: 10 * time.Second, ResponseHeaderTimeout: 30 * time.Second}
		httpClient := http.Client{Timeout: time.Second * 35, Transport: &httpTransport}
		proxyURL, ok := os.LookupEnv(EnvKeyBotHTTPProxyURL)
		if ok && proxyURL != "" {
			httpProxyURL, err := url.Parse(proxyURL)
			if nil != err {
				log.Fatal().Err(err).Msg("failed to parse bot http proxy url")
			}
			httpTransport.Proxy = http.ProxyURL(httpProxyURL)
		}

		opts := []bot.Option{
			bot.WithCheckInitTimeout(5 * time.Second),
			bot.WithHTTPClient(25*time.Second, &httpClient),
			bot.WithDefaultHandler(handler.handleMessage),
		}

		token, ok := os.LookupEnv(EnvKeyBotToken)
		if !ok {
			log.Fatal().Str("key", EnvKeyBotToken).Msg("required environment variable is not set")
		}

		b, err := bot.New(token, opts...)
		if nil != err {
			log.Fatal().Err(err).Msg("failed to initialize bot instance")
		}

		b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, handler.handleStartCommand)
		b.RegisterHandler(bot.HandlerTypeMessageText, "/info", bot.MatchTypeExact, handler.handleInfoCommand)
		b.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypeExact, handler.handleHelpCommand)
		b.Start(ctx)

		return nil
	}
}

type Handler struct {
	log               zerolog.Logger
	domainsFileName   string
	publishChatID     string
	supportUserChatID string
	db                *sql.DB
}

func extractDomainApexZone(msg string) (string, error) {
	parsedURL, err := url.Parse(strings.TrimSpace(msg))
	if nil != err {
		return "", err
	}

	domain := parsedURL.Host
	if domain == "" {
		path := parsedURL.Path
		parts := strings.SplitN(path, "/", 2)
		if len(parts) < 1 {
			return "", fmt.Errorf("could not extract domain from path '%s'", path)
		}
		domain = parts[0]
	}

	partsCount := strings.Count(domain, ".")
	if partsCount > 5 {
		return "", fmt.Errorf("subdomains depth exceeded maximum limit in '%s'", domain)
	}
	if partsCount < 1 {
		return "", fmt.Errorf("could not find domain apex zone and tld parts in '%s'", domain)
	}
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("could not extract domain apex zone from '%s'", domain)
	}
	apex, tld := parts[partsCount-1], parts[partsCount]

	return apex + "." + tld, nil
}

func (h *Handler) handleMessage(ctx context.Context, b *bot.Bot, update *models.Update) {
	if shouldDiscard(update) {
		return
	}
	// TODO: check if user can pass the daily rate limit
	log := h.loggerFromUpdate(update)

	domain, err := extractDomainApexZone(update.Message.Text)
	if nil != err {
		log.
			Debug().
			Err(err).
			Msg("failed to extract domain from message text")
		return
	}
	log = log.With().Str("domain", domain).Logger()

	successMessageText := "`" + domain + "`"
	replyMsg := bot.SendMessageParams{
		ChatID:    h.publishChatID,
		Text:      successMessageText,
		ParseMode: ParseModeMarkdownV1,
	}
	if _, err := b.SendMessage(ctx, &replyMsg); nil != err {
		log.
			Error().
			Err(err).
			Dict("reply_message", zerolog.Dict().
				Str("chat_id", h.publishChatID).
				Str("text", successMessageText),
			).
			Msg("failed to send message containing domain to publish chat")
	}

	file, err := os.OpenFile(h.domainsFileName, os.O_APPEND|os.O_CREATE|os.O_SYNC|os.O_WRONLY, 0644)
	if nil != err {
		log.
			Error().
			Err(err).
			Str("domains_filename", h.domainsFileName).
			Msg("failed to open domains file")
		h.informSupport(ctx, b, err)
		return
	}
	defer func() {
		if err := file.Close(); nil != err {
			log.Error().Err(err).Msg("failed to close domains file")
		}
	}()

	textToWrite := domain + "\n"
	if n, err := file.WriteString(textToWrite); nil != err {
		log.
			Error().
			Err(err).
			Str("filename", file.Name()).
			Str("text_to_write", textToWrite).
			Str("domains_filename", h.domainsFileName).
			Msg("failed to write domain to domains file")
		h.informSupport(ctx, b, err)
		return
	} else if expectedBytes := len(textToWrite); n != expectedBytes {
		log.
			Warn().
			Str("filename", file.Name()).
			Int("written_bytes_no", n).
			Int("expected_bytes_no", expectedBytes).
			Str("text_to_write", textToWrite).
			Str("domains_filename", h.domainsFileName).
			Msg("unexpected number of bytes was written to the domains file")
		h.informSupport(ctx, b, err)
		return
	}

	// TODO: increment user today's usage by one

	chatID := update.Message.Chat.ID
	replyMsg = bot.SendMessageParams{
		ChatID:           chatID,
		ReplyToMessageID: update.Message.ID,
		Text:             successMessageText,
		ParseMode:        ParseModeMarkdownV1,
	}
	if _, err := b.SendMessage(ctx, &replyMsg); nil != err {
		log.
			Error().
			Err(err).
			Dict("reply_message", zerolog.Dict().
				Int64("chat_id", chatID).
				Str("text", successMessageText),
			).
			Msg("failed to send success reply message to user chat")
		return
	}
}

func (h *Handler) informSupport(ctx context.Context, b *bot.Bot, err error) {
	chatID := h.supportUserChatID
	msg := bot.SendMessageParams{
		ChatID:    chatID,
		Text:      "An unexpected error occurred. Please check the logs...\n\n```\n" + err.Error() + "```",
		ParseMode: ParseModeMarkdownV1,
	}
	if _, sendErr := b.SendMessage(ctx, &msg); nil != sendErr {
		h.log.
			Error().
			Err(sendErr).
			AnErr("root_error", err).
			Dict("reply_message", zerolog.Dict().
				Str("chat_id", chatID),
			).
			Msg("failed to send message to support user chat")
		return
	}
}

func (h *Handler) handleStartCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	if shouldDiscard(update) {
		return
	}
	log := h.loggerFromUpdate(update)

	replyText := strings.Join(
		[]string{
			fmt.Sprintf("Compiled At: `%s`", bot.EscapeMarkdown(AppCompileTime)),
			fmt.Sprintf("Version: `%s`", bot.EscapeMarkdown(AppVersion)),
		},
		"\n",
	)
	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      replyText,
		ParseMode: models.ParseModeMarkdown,
	}); nil != err {
		log.Error().Err(err).Str("reply_text", replyText).Msg("failed to send start command success reply message")
	}
}

//go:embed info.txt
var infoCommandReplyMessageText string

func (h *Handler) handleInfoCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	if shouldDiscard(update) {
		return
	}
	log := h.loggerFromUpdate(update)

	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   infoCommandReplyMessageText,
	}); nil != err {
		log.Error().Err(err).Msg("failed to send info command success reply message")
	}
}

//go:embed help.txt
var helpCommandReplyMessageText string

func (h *Handler) handleHelpCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	if shouldDiscard(update) {
		return
	}
	log := h.loggerFromUpdate(update)

	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      helpCommandReplyMessageText,
		ParseMode: ParseModeMarkdownV1,
	}); nil != err {
		log.Error().Err(err).Msg("failed to send help command success reply message")
	}
}
