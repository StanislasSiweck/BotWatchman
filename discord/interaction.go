package discord

import (
	"bot-serveur-info/serveur"
	"bot-serveur-info/sql"
	"github.com/bwmarrin/discordgo"
	"log"
	"time"
)

var AllServers = map[string]sql.Server{}
var Mes *discordgo.Message

var (
	commandsHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"server add": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			addServerCommand(s, i)
		},
		"server remove": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			removeServerCommand(s, i)
		},
	}
	componentsHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"update": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			ServerInfo()
			response := &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Refreshed",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			}
			_ = s.InteractionRespond(i.Interaction, response)
		},
	}
)

func InteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {

	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		data := i.ApplicationCommandData()
		command := data.Name

		if len(data.Options) == 1 {
			command = data.Name + " " + data.Options[0].Name
		}
		if h, ok := commandsHandlers[command]; ok {
			h(s, i)
		}
	case discordgo.InteractionMessageComponent:
		if h, ok := componentsHandlers[i.MessageComponentData().CustomID]; ok {
			h(s, i)
		}
	}
}

func addServerCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	appOption := i.ApplicationCommandData().Options[0]
	server := sql.Server{
		IP:   appOption.Options[0].StringValue(),
		Port: appOption.Options[1].StringValue(),
	}

	if err := sql.AddServer(server); err != nil { // Create the server in the database
		log.Println(err)
	}

	AllServers[server.IP+":"+server.Port] = server // Add to local list

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{ // Send response to Discord
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: "Server added",
		},
	})
	if err != nil {
		log.Println(err)
	}
}

func removeServerCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	appOption := i.ApplicationCommandData().Options[0]
	delete(AllServers, appOption.Options[0].StringValue()+":"+appOption.Options[1].StringValue()) // Remove from local list

	ip, port := appOption.Options[0].StringValue(), appOption.Options[1].StringValue()

	if err := sql.RemoveServer(ip, port); err != nil { // Remove from database
		log.Println(err)
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{ // Send response to Discord
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: "Server removed",
		},
	})
	if err != nil {
		log.Println(err)
	}
}

func RefreshServerInfo() {
	for {
		ServerInfo()
		time.Sleep(1 * time.Minute)
	}
}

func ServerInfo() {
	var Fields []*discordgo.MessageEmbedField
	for _, server := range AllServers {

		info, err := serveur.GetServerInfo(server)
		if err != nil {
			log.Println(err)
			continue
		}

		Field := serveur.CreateField(info, server)

		// adds the new field to the Fields slice
		Fields = append(Fields, Field)
	}

	// edit a Discord message with the specified fields
	content := ""
	messageEdit := discordgo.MessageEdit{
		Content: &content,
		ID:      Mes.ID,
		Channel: Mes.ChannelID,
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Emoji: discordgo.ComponentEmoji{
							Name: "🔄",
						},
						Style:    discordgo.PrimaryButton,
						CustomID: "update",
					},
				},
			},
		},
		Embed: &discordgo.MessageEmbed{
			Title:       "Server watch list",
			Description: "━━━━━━━━━━━━━━━━━━━━━━━━",
			Color:       0x5ad65c,
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Update",
			},
			Timestamp: time.Now().Format(time.RFC3339),
			Fields:    Fields,
		},
	}
	_, err := DG.ChannelMessageEditComplex(&messageEdit)
	if err != nil {
		log.Fatal(err)
	}
}
