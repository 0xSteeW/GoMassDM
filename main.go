package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var Chunk *discordgo.GuildMembersChunk

func input(reader *bufio.Reader) string {
	raw, err := reader.ReadString('\n')
	if err != nil {
		log.Println("Something went wrong when reading stdin: " + err.Error())
		return ""
	}
	switch runtime.GOOS {
	case "windows":
		return strings.Replace(raw, "\r\n", "", -1)
	default:
		return strings.Replace(raw, "\n", "", 1)
	}
}

func init() {
	fmt.Println("Welcome to MassDM")
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Please provide your token:")
	token := input(reader)

	client, err := discordgo.New(token)
	if err != nil {
		log.Println("Could not open session (wrong token maybe?)")
		return
	}
	// Attempt to open session if no errors have ocurred
	client.AddHandler(guildMembersChunk)
	err = client.Open()
	if err != nil {
		log.Println("Could not open session: " + err.Error())
		return
	}

	// Now we are in the session
	currentUser, _ := client.User("@me")
	fmt.Println("Hello, ", currentUser.String(), "!")
	fmt.Println("Please specify the message content you want to send. (Dont use new lines!)")
	message := input(reader)
	confirm(reader)
	fmt.Println("Do you want to exclude any server? Please separate each one with a comma. You can leave this field empty.")
	ignore := input(reader)
	var ignoreGuilds []*discordgo.Guild
	if ignore != "" {
		guildIDS := strings.Split(ignore, ",")
		for _, guildID := range guildIDS {
			guildID = strings.TrimSpace(guildID)
			guild, err := client.Guild(guildID)
			if err == nil {
				ignoreGuilds = append(ignoreGuilds, guild)
			}
		}
	}
	fmt.Println("I will ignore these guilds: " + formatGuildsToString(ignoreGuilds))
	fmt.Println("Proceeding to MassDM")

	// Call main DM function
	massDM(client, ignoreGuilds, message)
}

func guildMembersChunk(client *discordgo.Session, chunk *discordgo.GuildMembersChunk) {
	fmt.Println("RECEIVED CHUNK")
	Chunk = chunk
}

// Main DM function
func massDM(client *discordgo.Session, ignore []*discordgo.Guild, message string) {
	log.Println("Getting all users...")
	users := getAllUsers(client, ignore)

	for _, user := range users {
		fmt.Println("Attempting to DM: " + user.String())
		channelDM, err := client.UserChannelCreate(user.ID)
		if err != nil {
			log.Println("Could not create DM channel with: " + user.String())
			continue
		}
		_, err = client.ChannelMessageSend(channelDM.ID, message)
		if err != nil {
			log.Println("Could not DM: " + user.String())
			continue
		}
		fmt.Println("DM'd " + user.String() + " successfully.")
	}
}

func checkRepeated(user *discordgo.User, users []*discordgo.User) bool {
	for _, userInList := range users {
		if user.ID == userInList.ID {
			return true
		}
	}
	return false
}

func getAllGuilds(client *discordgo.Session) []*discordgo.Guild {
	return client.State.Guilds
}

func membersToUsers(members []*discordgo.Member) []*discordgo.User {
	var userList []*discordgo.User
	for _, member := range members {
		userList = append(userList, member.User)
	}
	return userList
}

func checkIgnore(guild *discordgo.Guild, guilds []*discordgo.Guild) bool {
	for _, guildInList := range guilds {
		if guildInList.ID == guild.ID {
			return true
		}
	}
	return false
}

func getAllUsers(client *discordgo.Session, ignore []*discordgo.Guild) []*discordgo.User {
	log.Println("Getting all guilds...")
	var massList []*discordgo.User
	var guildUsers []*discordgo.User
	allGuilds := getAllGuilds(client)
	for _, guild := range allGuilds {
		log.Println("Working with: " + guild.Name)
		// Ignore this guild
		// if checkIgnore(guild, allGuilds) {
		//	continue
		// }
		// guildUsers = batchUserList(client, guild.ID)
		// members, err := client.GuildMembers(guild.ID, "", 5)
		/*if err != nil {
			log.Println("Could not get users of guild: " + guild.Name)
			log.Println(err)
			continue
		}
		guildUsers = membersToUsers(members)*/
		guildUsers = membersToUsers(guild.Members)
		// guildUsers = membersToUsers(guild.Members)
		log.Println("This server has: " + strconv.Itoa(len(guildUsers)) + " users.")
		for _, user := range guildUsers {
			log.Println("Working with user: " + user.String())
			//if !checkRepeated(user, massList) && user.ID != client.State.User.ID && !user.Bot {
			if user.ID == client.State.User.ID {
				continue
			}
			log.Println("Added user to MassDM list: " + user.String())
			massList = append(massList, user)
			//}
		}
	}
	return massList
}

func batchUserList(client *discordgo.Session, guildID string) []*discordgo.User {
	err := client.RequestGuildMembers(guildID, "", 0)
	if err != nil {
		log.Println(err)
		return nil
	}
	if Chunk == nil {
		return nil
	}
	return membersToUsers(Chunk.Members)
}

func formatGuildsToString(guilds []*discordgo.Guild) string {
	var concat string
	for _, guild := range guilds {
		concat = concat + " " + guild.Name
	}
	return strings.TrimSpace(concat)
}

func confirm(reader *bufio.Reader) {
	fmt.Println("Are you sure? y/n")
	switch strings.ToLower(input(reader)) {
	case "y":
		return
	case "n":
		os.Exit(0)
		return
	default:
		confirm(reader)
	}
}
