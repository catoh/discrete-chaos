package main

import (
	"flag"
	"os"
	"os/signal"
	"log" 

	"github.com/bwmarrin/discordgo"
)

var (
	token = flag.String("token", "TM0NzY4NTczNjk2MjMyNjY3Mg.GSUhXf.ZA68rjzVbwFneBRMtywESct7IVETPmYmiB5X9Y", "Bot Token")
	appID = flag.String("appid", "1347685736962326672", "Application ID")
	guildID = flag.String("guildid", "1347343214067318885", "Test Guild ID")
)

func main() {
	flag.Parse()
	s, _ := discordgo.New("Bot " + *token)  
	_, err := s.ApplicationCommandBulkOverwrite(*appID, *guildID, []*discordgo.ApplicationCommand{
	  {
		Name:        "hello-world",
		Description: "Showcase of a basic slash command",
	  },
	})
	if err != nil {
	  // Handle the error
	}  s.AddHandler(func (
	  s *discordgo.Session,
	  i *discordgo.InteractionCreate,
	) {
	  data := i.ApplicationCommandData()
	  switch data.Name {
	  case "hello-world":
		err := s.InteractionRespond(
		  i.Interaction,
		  &discordgo.InteractionResponse {
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData {
			  Content: "Hello world!",
			},
		  },
		)
		if err != nil {
		  // Handle the error
		}
	  }
	})  err := s.Open()
	if err != nil {
		// Handle the error
	}  stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-stop
   
	err = s.Close()
	if err != nil {
	  // Handle the error
	}
  }