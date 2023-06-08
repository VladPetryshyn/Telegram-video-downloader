package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"os/signal"
	"regexp"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/joho/godotenv"
	"github.com/kkdai/youtube/v2"
	"mvdan.cc/xurls/v2"
)

func main() {
	err := godotenv.Load(".env")
	if err == nil {
		token := os.Getenv("TOKEN")
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()

		opts := []bot.Option{
			bot.WithDefaultHandler(handler),
		}

		b, err := bot.New(token, opts...)
		if err != nil {
			panic(err)
		}

		b.Start(ctx)
	}
}

func handler(ctx context.Context, b *bot.Bot, update *models.Update) {
	rxStrict := xurls.Relaxed()

	urls := rxStrict.FindAllString(update.Message.Text, -1)
	if len(urls) < 1 {
		return
	}

	youtubeLinks := []string{}

	for _, url := range urls {
		match, _ := regexp.MatchString(`(www.|)(youtube\.com|youtu\.be)\/(watch\?v=.+|.+)`, url)

		if match {
			youtubeLinks = append(youtubeLinks, url)
		}
	}

	client := youtube.Client{}

	for _, videoID := range youtubeLinks {
		video, err := client.GetVideo(videoID)
		if err != nil {
			panic(err)
		}

		formats := video.Formats.WithAudioChannels()

		for _, form := range formats {
			if form.Quality == "small" {
				downloadingMessage, err := b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "downloading",
				})

				stream, _, err := client.GetStream(video, &form)
				if err != nil {
					panic(err)
				}

				s, err := ioutil.ReadAll(stream)

				vid := &models.InputFileUpload{
					Data: bytes.NewBuffer(s),
				}

				readyToSend, err := b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Ready to send",
				})

				_, err = b.SendVideo(ctx, &bot.SendVideoParams{
					ChatID: update.Message.Chat.ID,
					Video:  vid,
				})

				if err != nil {
					b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: update.Message.Chat.ID,
						Text:   "Video is too big",
					})
				}

				b.DeleteMessage(ctx, &bot.DeleteMessageParams{
					ChatID:    update.Message.Chat.ID,
					MessageID: downloadingMessage.ID,
				})

				b.DeleteMessage(ctx, &bot.DeleteMessageParams{
					ChatID:    update.Message.Chat.ID,
					MessageID: readyToSend.ID,
				})

				break
			}
		}
	}

}
