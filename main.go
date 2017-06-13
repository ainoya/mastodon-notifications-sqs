package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"fmt"

	"golang.org/x/net/html"

	mastodon "github.com/ainoya/go-mastodon"
	"github.com/bluele/slack"

	yaml "gopkg.in/yaml.v2"
)

const settingFile = "./setting.yml"

var wh *slack.WebHook
var s Settings

// Settings from yaml
type Settings struct {
	ServerConfs     []ServerConf `yaml:"serverConfs"`
	SlackWebHookURL string       `yaml:"slackWebHookURL"`
}

// ServerConf mastodon server's setting
type ServerConf struct {
	ServerURL          string `yaml:"serverURL"`
	StreamingServerURL string `yaml:"streamingServerURL"`
	ClientID           string `yaml:"clientID"`
	ClientSecret       string `yaml:"clientSecret"`
	Account            string `yaml:"account"`
	Password           string `yaml:"password"`
}

// notification struct
type notification struct {
	displayName string
	content     string
	url         string
}

func main() {
	// load setting file
	buf, err := ioutil.ReadFile(settingFile)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(buf, &s)
	if err != nil {
		panic(err)
	}

	wh = slack.NewWebHook(s.SlackWebHookURL)

	wg := &sync.WaitGroup{}
	for i := range s.ServerConfs {
		log.Println("loop", i)
		wg.Add(1)
		go connect(s.ServerConfs[i])
	}

	// no reachable
	wg.Wait()
}

func connect(conf ServerConf) {
	c := mastodon.NewClient(&mastodon.Config{
		Server:          conf.ServerURL,
		StreamingServer: conf.StreamingServerURL,
		ClientID:        conf.ClientID,
		ClientSecret:    conf.ClientSecret,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := c.Authenticate(ctx, conf.Account, conf.Password)
	if err != nil {
		log.Println("authenticate")
		log.Fatal(err)
	}

	wsc := c.NewWSClient()

	q, err := wsc.StreamingWSPublic(context.Background(), false)
	if err != nil {
		log.Println("streamingwsuser")
		log.Fatal(err)
	}

	//	log.Println("start")

	cnl := make(chan bool)
	go watchStream(q, cnl)

	select {
	case <-cnl:
		// log.Println("channel down and restart")
	}
	connect(conf)
}

func watchStream(q chan mastodon.Event, c chan bool) {
	defer func() { c <- true }()
	// get event stream
	for e := range q {
		if t, ok := e.(*mastodon.UpdateEvent); ok {
			pushMessage(notification{
				displayName: t.Status.Account.Username,
				content:     textContent(t.Status.Content),
				url:         t.Status.URL,
			})
		}
	}
}

func textContent(s string) string {
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		return s
	}
	var buf bytes.Buffer

	var extractText func(node *html.Node, w *bytes.Buffer)
	extractText = func(node *html.Node, w *bytes.Buffer) {
		if node.Type == html.TextNode {
			data := strings.Trim(node.Data, "\r\n")
			if data != "" {
				w.WriteString(data)
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			extractText(c, w)
		}
		if node.Type == html.ElementNode {
			name := strings.ToLower(node.Data)
			if name == "br" {
				w.WriteString("\n")
			}
		}
	}
	extractText(doc, &buf)
	return buf.String()
}

// push a message to AWS SQS
func pushMessage(n notification) {
	wh.PostMessage(&slack.WebHookPostPayload{
		IconEmoji: ":elephant:",
		Attachments: []*slack.Attachment{
			{
				AuthorName: fmt.Sprintf("@%s", n.displayName),
				AuthorLink: fmt.Sprintf("%s/@%s", s.ServerConfs[0].ServerURL, n.displayName),
				Text:       n.content,
			},
		},
		Username: "makepadon",
	})
}

// remove xml tags
// mention event contains HTML tags
func removeTag(str string) string {
	rep := regexp.MustCompile(`<("[^"]*"|'[^']*'|[^'">])*>`)
	str = rep.ReplaceAllString(str, "")
	return str
}
