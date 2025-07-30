package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Bot parameters
type BotParameters struct {
	BOT_TOKEN string
	GUILD_ID  string
}

var (
	jsonData, _ = os.ReadFile("parameters.json")

	bp BotParameters
	_  = json.Unmarshal(jsonData, &bp)

	GuildID        = flag.String("guild", bp.GUILD_ID, "Test guild ID. If not passed - bot registers commands globally")
	BotToken       = flag.String("token", bp.BOT_TOKEN, "Bot access token")
	RemoveCommands = flag.Bool("rmcmd", true, "Remove all commands after shutdowning or not")
)

var s *discordgo.Session

func init() { flag.Parse() }

func init() {
	var err error
	s, err = discordgo.New("Bot " + *BotToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}
}

var (
	dmPermission                   = false
	defaultMemberPermissions int64 = discordgo.PermissionManageServer
	r                              = strings.NewReplacer(" ", "", "+", " + ", "-", " - ")

	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "droll",
			Description: "Roll a number of d10s and compare to a difficulty threshold, returns the number of successes",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "dice-pool",
					Description: "Number of dice in the pool",
					Required:    true,
				},

				// Required options must be listed first since optional parameters
				// always come after when they're used.
				// The same concept applies to Discord's Slash-commands API

				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "difficulty",
					Description: "Difficulty threshold for success",
					Required:    true,
				},
			},
		},
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"droll": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			// Access options in the order provided by the user.
			options := i.ApplicationCommandData().Options

			// Or convert the slice into a map
			optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
			for _, opt := range options {
				optionMap[opt.Name] = opt
			}

			// ready the pool and diffs for assignment later
			var pool int = 0
			var diff int = 0

			// This example stores the provided arguments in an []interface{}
			// which will be used to format the bot's response
			margs := make([]interface{}, 0, len(options))
			msgformat := ""

			// Get the value from the option map.
			// When the option exists, ok = true
			if opt, ok := optionMap["dice-pool"]; ok {
				// Option values must be type asserted from interface{}.
				// Discordgo provides utility functions to make this simple.
				pool = eval(r.Replace(opt.StringValue()))
				margs = append(margs, pool)
				msgformat += "> dice-pool: %d\n"
			}

			if opt, ok := optionMap["difficulty"]; ok {
				diff = eval(r.Replace(opt.StringValue()))
				margs = append(margs, diff)
				msgformat += "> difficulty: %d\n"
			}

			msgformat += roll(pool, diff)

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				// Ignore type for now, they will be discussed in "responses"
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf(
						msgformat,
						margs...,
					),
				},
			})
		},
	}
)

func roll(p int, d int) string {
	var result string = "You rolled: "
	var rolls string = ""
	sum := 0
	reroll_count := 0
	for i := 0; i < p; i++ {
		// roll d10
		roll := rand.IntN(10) + 1
		// append roll to pool message
		if rolls != "" {
			rolls += ", "
		}
		rolls += strconv.Itoa(roll)
		// compare roll to difficulty
		if roll >= d {
			// add to sum if success
			sum += 1
			// if a 10 was rolled, add another success
			if roll == 10 {
				sum += 1
			}
		} else if roll == 1 {
			// or reduce sum if roll = 1
			// and explode 1s
			reroll_count -= 1
			explode := "("
			for x := 0; x <= 1; {
				x = rand.IntN(10) + 1
				sum -= 1
				reroll_count += 1
				explode += strconv.Itoa(x) + ","
			}
			if explode != "(" {
				explode = strings.TrimSuffix(explode, ",")
			}
			explode += ")"
			rolls += explode
		}

	}
	result += rolls + "\n"
	result += "Total Successes: " + strconv.Itoa(sum) + "\n"
	if reroll_count > 0 {
		result += "Exploded " + strconv.Itoa(reroll_count) + " times\n"
	}
	return result
}

func eval(term string) int {
	var formula = strings.Split(term, " ")
	var add = false
	var sub = false
	var result = 0
	for _, v := range formula {
		if v == "+" {
			add = true
		} else if v == "-" {
			sub = true
		} else {
			num, _ := strconv.Atoi(v)
			if add {
				result += num
				add = false
			} else if sub {
				result -= num
				sub = false
			} else {
				result = num
			}
		}
	}
	return result
}

func init() {
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
}

func main() {
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})
	err := s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}

	log.Println("Adding commands...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, *GuildID, v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	defer s.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-stop

	if *RemoveCommands {
		log.Println("Removing commands...")
		// // We need to fetch the commands, since deleting requires the command ID.
		// // We are doing this from the returned commands on line 375, because using
		// // this will delete all the commands, which might not be desirable, so we
		// // are deleting only the commands that we added.
		// registeredCommands, err := s.ApplicationCommands(s.State.User.ID, *GuildID)
		// if err != nil {
		// 	log.Fatalf("Could not fetch registered commands: %v", err)
		// }

		for _, v := range registeredCommands {
			err := s.ApplicationCommandDelete(s.State.User.ID, *GuildID, v.ID)
			if err != nil {
				log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
			}
		}
	}

	log.Println("Gracefully shutting down.")
}
