package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// This contains all the core functions - it is designed so it can be copied into any other project to create new agents that can receive/send back to the original agent or to any other program through json input/string output

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
)

type promptDefinition struct {
	Name        string
	Description string
	Parameters  string
}

var today = time.Now().Format("January 2, 2006")

var defaultprompt = promptDefinition{
	Name:        "Default",
	Description: "Default Prompt",
	Parameters:  "You are a helpful assistant. Please generate truthful, accurate, and honest responses while also keeping your answers succinct and to-the-point. Today's date is: " + today,
}

type Agent struct {
	prompt     promptDefinition
	tokencount int
	api_key    string
	model      string
	Messages   []Message
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type RequestBody struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	TotalTokens      int `json:"total_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

var homeDir string // Home directory for storing agent files/folders /Prompts /Functions /Saves
var guiFlag bool = false
var consoleFlag bool = false
var savechatName string

// var model string = "gpt-3.5-turbo"
var defaultmodel string = "mistral-small"
var callcost float64 = 0.002
var maxtokens int = 2048

var authstring string
var allowedIps []string
var allowAllIps bool = false
var port string = ":4177"

func (agent *Agent) getmodelURL() string {
	// to be expanded
	var url string
	switch {
	case strings.HasPrefix(agent.model, "mistral"):
		url = "https://api.mistral.ai/v1/chat/completions"
	case strings.HasPrefix(agent.model, "gpt"):
		url = "https://api.openai.com/v1/chat/completions"
	default:
		// handle invalid model here
		fmt.Println("Error: Invalid model")
	}
	return url
}

func newAgent(key ...string) Agent {
	agent := Agent{}
	agent.prompt = defaultprompt
	agent.setprompt()
	agent.model = defaultmodel
	agent.tokencount = 0
	agent.getflags()
	if agent.api_key == "" {
		if len(key) == 0 {
			panic("Enter key with -key flag!")
		}
	}
	return agent
}

func (agent *Agent) getflags() {
	// Set default home dir
	homeDir, _ = gethomedir()
	if homeDir != "" {
		homeDir = filepath.Join(homeDir, "AgentSmith")
	}

	// range over args to get flags
	for index, flag := range os.Args {
		var arg string
		if index < len(os.Args)-1 {
			item := os.Args[index+1]
			if !strings.HasPrefix(item, "-") {
				arg = item
			}
		}

		switch flag {
		case "-key":
			// Set API key
			agent.api_key = arg
		case "-home":
			// Set home directory
			homeDir = arg
		case "-save":
			// chats save to homeDir/Saves
			savechatName = arg
		case "-load":
			// load chat from homeDir/Saves
			agent.loadfile("Chats", arg)
		case "-prompt":
			// Set prompt
			agent.setprompt(arg)
		case "-model":
			// Set model
			defaultmodel = arg
		case "-maxtokens":
			// Change setting variable
			maxtokens, _ = strconv.Atoi(arg)
		case "-message":
			// Get the argument after the flag]
			// Set messages for the agent/create chat history
			agent.setmessage(RoleUser, arg)
		case "-messageassistant":
			// Allows multiple messages with different users to be loaded in order
			agent.setmessage(RoleAssistant, arg)
		case "--gui":
			// Run GUI
			guiFlag = true
		case "-ip":
			// allow ip
			if arg == "all" {
				allowAllIps = true
			} else {
				allowedIps = append(allowedIps, arg)
			}
		case "-auth":
			authstring = arg
		case "-port":
			// change port
			port = ":" + arg
		case "-allowallips":
			// allow all ips
			fmt.Println("Warning: Allowing all incoming connections.")
			allowAllIps = true
		case "--console":
			// Run as console
			consoleFlag = true
		}
	}
}

func gettextinput() string {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input := scanner.Text()
		if len(input) == 0 {
			return ""
		}
		return input
	}
	return ""
}

func (agent *Agent) reset() {
	*agent = newAgent()
	callcost = 0.002
	maxtokens = 2048
}

func (agent *Agent) setmessage(role, content string) {
	message := Message{
		Role:    role,
		Content: content,
	}
	agent.Messages = append(agent.Messages, message)
}

func (agent *Agent) setprompt(prompt ...string) {
	agent.Messages = []Message{}
	if len(prompt) == 0 {
		agent.setmessage(RoleSystem, agent.prompt.Parameters)
	} else {
		agent.setmessage(RoleSystem, prompt[0])
	}
	agent.tokencount = 0
}

func (agent *Agent) getresponse() (Message, error) {
	var response Message

	// Create the request body
	requestBody := &RequestBody{
		Model:    agent.model,
		Messages: agent.Messages,
	}

	// Encode the request body as JSON
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Println("Error encoding request body:", err)
		return response, err
	}

	// Create the HTTP request
	req, err := http.NewRequest(http.MethodPost, agent.getmodelURL(), bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error creating HTTP request:", err)
		return response, err
	}

	// Set the request headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", agent.api_key))

	// Send the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending HTTP request:", err)
		return response, err
	}

	// Handle the HTTP response
	defer resp.Body.Close()

	// Decode the response body into a Message object
	var chatresponse ChatResponse
	err = json.NewDecoder(resp.Body).Decode(&chatresponse)
	if err != nil {
		fmt.Println("Error decoding JSON response:", err)
		return response, err
	}

	if len(chatresponse.Choices) == 0 {
		fmt.Println("Error with response:", chatresponse)
		return response, err
	}

	fmt.Println(chatresponse)

	// Print the decoded message
	fmt.Println("Decoded message:", chatresponse.Choices[0].Message.Content)

	agent.tokencount = chatresponse.Usage.TotalTokens

	// Add message to chain for Agent
	agent.Messages = append(agent.Messages, chatresponse.Choices[0].Message)

	return chatresponse.Choices[0].Message, nil
}

func gethomedir() (string, error) {
	for _, item := range os.Args {
		if item == "-homedir" {
			homeDir = item
		} else {
			usr, err := user.Current()
			if err != nil {
				fmt.Println("Failed to get current user:", err)
				return "", err
			}

			// Retrieve the path to user's home directory
			homeDir = usr.HomeDir
		}
	}
	return homeDir, nil
}

func getrequest() map[string]string {
	// receive request from assistant
	// receives {"key": "string"} argument and converts it to map[string]string
	var req map[string]string
	args := os.Args[1]
	_ = json.Unmarshal([]byte(args), &req)
	return req
}

func getsubrequest(input string) map[string]string {
	// receives request from another function
	// receives {"key": "string"} argument and converts it to map[string]string
	var req map[string]string
	args := input
	_ = json.Unmarshal([]byte(args), &req)
	return req
}

func (agent *Agent) savefile(data interface{}, filetype string, input ...string) (string, error) {
	// savetype must be Chats, Prompts, or Functions

	var filename string
	if len(input) == 0 {
		currentTime := time.Now()
		filename = currentTime.Format("20060102150405")
	} else {
		filename = strings.Replace(input[0], " ", "_", -1)
	}

	var filedir string
	if strings.HasSuffix(filename, ".json") {
		filedir = filepath.Join(homeDir, filetype, filename)
	} else {
		filedir = filepath.Join(homeDir, filetype, filename+".json")
	}
	appDir := filepath.Join(homeDir, filetype)
	err := os.MkdirAll(appDir, os.ModePerm)
	if err != nil {
		fmt.Println("Failed to create app directory:", err)
		return "", err
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	file, err := os.OpenFile(filedir, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = file.Write(jsonData)
	if err != nil {
		return "", err
	}

	return filedir, nil
}

func (agent *Agent) loadfile(filetype string, filename string) ([]byte, error) {

	var filedir string
	if strings.HasSuffix(filename, ".json") {
		filedir = filepath.Join(homeDir, filetype, filename)
	} else {
		filedir = filepath.Join(homeDir, filetype, filename+".json")
	}

	file, err := os.Open(filedir)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	switch filetype {
	case "Chats":
		agent.reset()
		newmessages := []Message{}
		err = json.Unmarshal(data, &newmessages)
		if err != nil {
			return nil, err
		}
		agent.Messages = newmessages
		return nil, err
	case "Functions":
		return data, nil
	case "Prompts":
		return data, nil
	}
	return nil, nil
}

func deletefile(filetype, filename string) error {
	var filedir string
	if strings.HasSuffix(filename, ".json") {
		filedir = filepath.Join(homeDir, filetype, filename)
	} else {
		filedir = filepath.Join(homeDir, filetype, filename+".json")
	}

	err := os.Remove(filedir)
	if err != nil {
		fmt.Println("Error deleting file:", err)
		return err
	}

	fmt.Println("File deleted successfully.")

	return nil
}

func getsavefilelist(filetype string) ([]string, error) {
	// Create a directory for your app
	savepath := filepath.Join(homeDir, filetype)
	files, err := os.ReadDir(savepath)
	if err != nil {
		return nil, err
	}
	var res []string

	fmt.Println("\nFiles:")

	for _, file := range files {
		filename := strings.ReplaceAll(file.Name(), ".json", "")
		res = append(res, filename)
		fmt.Println(file.Name())
	}

	return res, nil
}

func (agent *Agent) deletelines(editchoice string) error {
	// Use regular expression to find all numerical segments in the input string
	reg := regexp.MustCompile("[0-9]+")
	nums := reg.FindAllString(editchoice, -1)

	var sortednums []int
	// Convert each numerical string to integer and sort
	for _, numStr := range nums {
		num, err := strconv.Atoi(numStr)
		if err != nil {
			return err
		}
		sortednums = append(sortednums, num)
	}

	sort.Ints(sortednums)

	fmt.Println("Deleting lines: ", sortednums)

	// go from highest to lowest to not fu the order
	for i := len(sortednums) - 1; i >= 0; i-- {
		agent.Messages = append(agent.Messages[:sortednums[i]], agent.Messages[sortednums[i]+1:]...)
	}

	return nil
}
