/*
TODO : Main fixes to do
- fix so it works in multiple servers at once(Streaming audio)
*/
package main

import (
	"Gobot/commands"
	"Gobot/music"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatal("Error loading .env file")
	}
	Token := os.Getenv("discord_token")
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}
	go dg.AddHandler(botStatus)
	go dg.AddHandler(messageCreate)
	go dg.AddHandler(slashCommands)
	go dg.AddHandler(interactionHandler)
	dg.Identify.Intents = discordgo.IntentsAll

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}
	// Registering commands here
	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "ping",
		Description: "Check the bot's latency to the Discord API",
	})
	if err != nil {
		fmt.Println("Cannot create slash command:", err)
	}

	// Next command
	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "play",
		Description: "Plays a YouTube or finds a youtube video if no url given",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "query",
				Description: "The URL of Youtube or name of the song to play",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			},
		},
	})

	if err != nil {
		fmt.Println("error creating slash command,", err)
		return
	}
	// Next command here
	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "test",
		Description: "This is a test command, does nothing if ran",
	})
	if err != nil {
		fmt.Println("Cannot create slash command:", err)
	}
	// Next command here
	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "fox",
		Description: "Shows a random fox using randomfox.ca api",
	})
	if err != nil {
		fmt.Println("Cannot create slash command:", err)
	}
	// Next command here
	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "skip",
		Description: "Skip the current song",
	})
	if err != nil {
		fmt.Println("Cannot create slash command:", err)
	}
	// Next command here
	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "cat",
		Description: "Shows a random cat",
	})
	if err != nil {
		fmt.Println("Cannot create slash command:", err)
	}
	// Next command here
	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "dog",
		Description: "Shows a random dog",
	})
	if err != nil {
		fmt.Println("Cannot create slash command:", err)
	}
	// Next command here
	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "save",
		Description: "Saves the current song to a DM",
	})
	if err != nil {
		fmt.Println("Cannot create slash command:", err)
	}
	// Next command here
	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "nowplaying",
		Description: "Shows the current song being played",
	})
	if err != nil {
		fmt.Println("Cannot create slash command:", err)
	}
	// Next command here
	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "queue",
		Description: "Shows the current queue",
	})
	if err != nil {
		fmt.Println("Cannot create slash command:", err)
	}
	// Next command here
	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "stop",
		Description: "Stops the music and clears the queue",
	})
	if err != nil {
		fmt.Println("Cannot create slash command:", err)
	}
	// Next command here
	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "help",
		Description: "Shows all avaliable commands",
	})
	if err != nil {
		fmt.Println("Cannot create slash command:", err)
	}
	// Next command here
	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "playnext",
		Description: "Add Youtube or Soundcloud song to the start of the queue",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "url",
				Description: "The URL of Youtube or Soundcloud audio",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			},
		},
	})

	if err != nil {
		fmt.Println("error creating slash command,", err)
		return
	}
	// Next command here
	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "prikol",
		Description: "Returns a random prikol(fact) :)",
	})

	if err != nil {
		fmt.Println("error creating slash command,", err)
		return
	}
	// Next command here
	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "song",
		Description: "View data about a song in the queue(Including those that /queue does not show)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "song_index",
				Description: "Index of the song in the queue",
				Type:        discordgo.ApplicationCommandOptionInteger,
				Required:    true,
			},
		},
	})
	if err != nil {
		fmt.Println("error creating slash command,", err)
		return
	}
	// Next command here
	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "remove",
		Description: "Remove a song from the queue",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "song_index",
				Description: "Index of the song in the queue",
				Type:        discordgo.ApplicationCommandOptionInteger,
				Required:    true,
			},
		},
	})
	if err != nil {
		fmt.Println("error creating slash command,", err)
		return
	}
	// Next command here
	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "d20",
		Description: "Gives a pseudorandom number for D20",
	})
	// Next command here
	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "d10",
		Description: "Gives a pseudorandom number for D10",
	})
	// Next command here
	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "rnd",
		Description: "Gives a pseudorandom number in [0,max_number) format",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "count",
				Description: "How many times to repeat",
				Type:        discordgo.ApplicationCommandOptionInteger,
				Required:    true,
			},
			{
				Name:        "max_number",
				Description: "Max number to use in the function",
				Type:        discordgo.ApplicationCommandOptionInteger,
				Required:    true,
			},
			{
				Name:        "dnd",
				Description: "Makes so 0 isnt possible(For dnd)(adds +1 to the end result)",
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Required:    false,
			},
		},
	})
	// Finished registering commands here

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	log.Println("Graceful shutdown")

	// Cleanly close down the Discord session.
	dg.Close()
}
func botStatus(s *discordgo.Session, r *discordgo.Ready) {
	err := s.UpdateCustomStatus("/help seems like a cool command")
	if err != nil {
		fmt.Printf("Error setting status: %v\n", err)
	}
}
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}
	// This is an immutable part of the code it will always be here >:D
	if strings.Contains(strings.ToLower(m.Content), "let us") {
		s.ChannelMessageSend(m.ChannelID, "https://imgs.search.brave.com/7GZQCq7OtNZZYRiVNvbJ8fiAZPYI4-TyUx6aYMbfPFg/rs:fit:860:0:0:0/g:ce/aHR0cHM6Ly90NC5m/dGNkbi5uZXQvanBn/LzA0LzYzLzg1Lzc5/LzM2MF9GXzQ2Mzg1/NzkzM19qaUJ6cjJq/eWh0emNCOXdJczJo/elh6MUplSkw2M3pq/dC5qcGc")
	}
	if m.Author.ID == os.Getenv("owner") && strings.Contains(strings.ToLower(m.Content), "!say") {
		s.ChannelMessageDelete(m.ChannelID, m.ID)
		content := strings.Split(m.Content, "!say")[1]
		s.ChannelMessageSend(m.ChannelID, content)
	}
}

func slashCommands(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	switch i.ApplicationCommandData().Name {
	case "ping":
		go commands.PingCommand(s, i)
	case "tiktok":
		urlOption := i.ApplicationCommandData().Options[0].StringValue()
		go commands.Stealshorts(s, urlOption, i)
	case "play":
	/*	url := i.ApplicationCommandData().Options[0].StringValue()
		go music.PlayAudio(s, i, url) */
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "play command for now is broken due to lack of E2EE/DAVE protocol, wait when im not lazy to fix it",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})

			if err != nil {
				fmt.Println("Error sending interaction response:", err)
				return
			}
	case "fox":
		go commands.RandomFox(s, i)
	case "skip":
		go music.Skip(s, i, i.GuildID)
	case "cat":
		go commands.RandomCat(s, i)
	case "dog":
		go commands.RandomDog(s, i)
	case "save":
		go music.Save(s, i)
	// case "test":
	// 	go commands.D10(s, i)
	case "d20":
		go commands.D20(s, i)
	case "d10":
		go commands.D10(s, i)
	case "rnd":
		count := i.ApplicationCommandData().Options[0]
		if count.Type != discordgo.ApplicationCommandOptionInteger {
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Count must be a number",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			if err != nil {
				log.Printf("ERROR: Unable to send wrong type response in rnd\n%s", err)
			}
			return
		}
		max_range := i.ApplicationCommandData().Options[1]
		dnd := len(i.ApplicationCommandData().Options) > 2 && i.ApplicationCommandData().Options[2].BoolValue()
		go commands.RandomNumber(s, i, int(count.IntValue()), int64(max_range.IntValue()), dnd)
	case "nowplaying":
		go music.NowPlaying(s, i)
	case "queue":
		go music.Queue(s, i, i.GuildID)
	case "stop":
		go music.Stop(s, i, i.GuildID)
	case "help":
		go commands.Help(s, i)
	case "playnext":
		url := i.ApplicationCommandData().Options[0].StringValue()
		go music.Playnext(s, i, url)
	case "prikol":
		go commands.RandomFact(s, i)
	case "remove":
		option := i.ApplicationCommandData().Options[0]
		if option.Type != discordgo.ApplicationCommandOptionInteger {
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Index must be a number",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})

			if err != nil {
				fmt.Println("Error sending interaction response:", err)
				return
			}
			return
		}
		index := option.IntValue()
		go music.Remove(s, i, i.GuildID, index)
	case "song":
		option := i.ApplicationCommandData().Options[0]
		if option.Type != discordgo.ApplicationCommandOptionInteger {
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Index must be a number",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})

			if err != nil {
				fmt.Println("Error sending interaction response:", err)
				return
			}
			return
		}
		index := option.IntValue()
		go music.SongData(s, i, i.GuildID, index)
	default:
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Why are you running /test command, I told you that it does nothing",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})

		if err != nil {
			fmt.Println("Error sending interaction response:", err)
			return
		}
	}
}

func interactionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionMessageComponent {
		return
	}
	switch i.MessageComponentData().CustomID {
	case "skip_button":
		music.Skip(s, i, i.GuildID)
	case "save_button":
		music.Save(s, i)
	case "queue_button":
		music.Queue(s, i, i.GuildID)
	default:
		fmt.Println("Unknown button pressed:", i.MessageComponentData().CustomID)
	}
}
