package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var Chunk []*discordgo.Member

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
	fmt.Println("Please input characters file (leave blank for default)")
	path := input(reader)
	fmt.Println("Proceeding to MassDM")

	// Call main DM function
	massDM(client, ignoreGuilds, message, path)
}

func guildMembersChunk(client *discordgo.Session, chunk *discordgo.GuildMembersChunk) {
	fmt.Println("Received chunk... ", len(chunk.Members), " users.")
	Chunk = append(Chunk, chunk.Members...)
}

// Main DM function
func massDM(client *discordgo.Session, ignore []*discordgo.Guild, message string, path string) {
	log.Println("Getting all users...")
	users := getAllUsers(client, ignore, path)

	var count int
	for _, user := range users {
		log.Println("Cooldown... ")
		time.Sleep(4 * time.Second)
		fmt.Println("Attempting to DM: "+user.String(), count, "/", len(users))
		count += 1
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
	if len(users) == 0 {
		return false
	}
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

func getLetters(path string) []string {
	var file *os.File
	var err error
	fmt.Println("Reading path:", path)
	if path == "" {
		path = "./characters.txt"
	}
	file, err = os.Open(path)
	defer file.Close()
	if err != nil {
		log.Println("Characters file not valid.")
		time.Sleep(5)
		os.Exit(0)
	}

	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println("Error reading file: ", err)
		time.Sleep(5)
		os.Exit(0)
	}
	chars := strings.Split(string(content), "\n")
	var finalChars []string
	for _, char := range chars {
		finalChars = append(finalChars, strings.ReplaceAll(char, "\n", ""))
	}
	return finalChars
}

func getAllUsers(client *discordgo.Session, ignore []*discordgo.Guild, path string) []*discordgo.User {
	log.Println("Getting all guilds...")
	var massList []*discordgo.User
	var guildUsers []*discordgo.User
	allGuilds := getAllGuilds(client)
	for _, guild := range allGuilds {
		log.Println("Working with: " + guild.Name)
		// Ignore this guild
		if len(ignore) > 0 && checkIgnore(guild, allGuilds) {
			log.Println("Ignoring " + guild.Name)
			continue
		}
		for _, letter := range getLetters(path) {
			fmt.Println("Waiting for user chunk... Letter ", letter)
			batchUserList(client, guild.ID, strings.TrimSpace(letter))
			time.Sleep(1 * time.Second)
		}
		guildUsers = membersToUsers(Chunk)
		Chunk = nil
		log.Println("This server has: " + strconv.Itoa(len(guildUsers)) + " users.")
		for _, user := range guildUsers {
			log.Println("Working with user: " + user.String())
			if !checkRepeated(user, massList) && user.ID != client.State.User.ID && !user.Bot {
				log.Println("Added user to MassDM list: " + user.String())
				massList = append(massList, user)
			}
		}
	}
	return massList
}

// Hardcoded function to prevent discord from blocking requests

func batchUserList(client *discordgo.Session, guildID string, letter string) {
	fmt.Println("Asking discord for members of guild: " + guildID)
	err := client.RequestGuildMembers(guildID, letter, 0)
	if err != nil {
		log.Println(err)
		return
	}
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
