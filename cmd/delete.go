package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/spf13/cobra"
)

// DelPost Delete struct for a) holding state and b) attaching functions to that will use that state
type DelPost struct {
	user             string
	cKey             string
	cSecret          string
	aToken           string
	aSecret          string
	tClient          *twitter.Client
	configFile       string
	postid           int
	destroyParams    *twitter.StatusDestroyParams
	statusUnRTParams *twitter.StatusUnretweetParams
}

// delPostCmd represents the delPost command
var delPostCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a twitter post",
	Long:  `Delete the specified twitter post.`,
	Run: func(cmd *cobra.Command, args []string) {
		dp := newDelPost()
		dp.init(cmd)

		// Begin command logic
		dp.run()
	},
}

// Add the command to rootCmd and define its flags. This is called regardless of what command
// was passed on the command line
func init() {
	rootCmd.AddCommand(delPostCmd)

	delPostCmd.Flags().IntP("post-id", "p", 0, "The ID of the post to be deleted.")
	delPostCmd.Flags().StringP("config-file", "c", "", "Config file holding credentials [Default: $HOME/.gtrc]")
	delPostCmd.Flags().StringP("consumer-key", "k", "", "Twitter Consumer Key")
	delPostCmd.Flags().StringP("consumer-secret", "s", "", "Twitter Consumer Secret")
	delPostCmd.Flags().StringP("access-token", "T", "", "Twitter Access Token")
	delPostCmd.Flags().StringP("access-secret", "S", "", "Twitter Access Secret")

	// Commented examples of making a Boolean, an Int, and a String flag
	// Also, some examples of making a flag required and making one hidden
	//
	// delPostCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	// delPostCmd.Flags().IntP("count", "c", 10, "Help message for count")
	// delPostCmd.Flags().StringP("msg", "m", "Boo!", "Help message for msg")
	// delPostCmd.MarkFlagRequired("msg")
	// delPostCmd.Flags().BoolP("secretToggle", "X", false, "Help message for secretToggle")
	// delPostCmd.MarkHidden("secretToggle")
}

// If the command is run, this does initialization needed for it to start
func (dp *DelPost) init(cmd *cobra.Command) {
	fmt.Printf("Initializing DelPost\n")

	dp.postid, _ = cmd.Flags().GetInt("post-id")
	dp.configFile, _ = cmd.Flags().GetString("config-file")
	dp.cKey, _ = cmd.Flags().GetString("consumer-key")
	dp.cSecret, _ = cmd.Flags().GetString("consumer-secret")
	dp.aToken, _ = cmd.Flags().GetString("access-token")
	dp.aSecret, _ = cmd.Flags().GetString("access-secret")

	var cFName string
	var cFileExists bool

	// If they didn't set a config file, we'll see if we can try to find it in
	// their home directory
	if dp.configFile == "" {
		home := os.Getenv("HOME")
		cFName = fmt.Sprintf("%s/.gtrc", home)
		// Make sure the file exists...
		if _, err := os.Stat(cFName); errors.Is(err, os.ErrNotExist) {
			cFileExists = false
		} else {
			cFileExists = true
			dp.configFile = cFName
		}
	}

	if dp.user == "" || dp.cKey == "" || dp.cSecret == "" || dp.aToken == "" || dp.aSecret == "" {
		// If we didn't get these on the commandline, try reading the config file (if it exists)
		if cFileExists {
			dp.readConfigFile(cFName)
		} else {
			// And if the config file doesn't exist, error out.
			log.Fatal("No config file or credentials provided. Cannot continue.")
		}
	}
	dp.setUpTwitterClient()
}

// Create new DelPost to track cmd state
func newDelPost() *DelPost {
	return &DelPost{}
}

// Your setup is done. Here's where you start your business logic
func (dp *DelPost) run() {
	fmt.Printf("Here's where you'd be running things\n")
	// The timeline of folks I follow
	// timeline, _, err := dp.tClient.Timelines.HomeTimeline(&twitter.HomeTimelineParams{Count: 10})
	// Hopefully *my* timeline
	dp.destroyParams = &twitter.StatusDestroyParams{TweetMode: "extended"}
	dp.statusUnRTParams = &twitter.StatusUnretweetParams{TweetMode: "extended"}
	for {
		timeline, _, err := dp.tClient.Timelines.UserTimeline(&twitter.UserTimelineParams{
			Count:     100,
			TweetMode: "extended",
		})

		if err == nil {
			if len(timeline) > 0 {
				for i, tweet := range timeline {
					fmt.Printf("[%d] {id: %d}", i, tweet.ID)
					if tweet.Retweeted {
						fmt.Printf("\n\t{%t} Timeline retweet: '%s'\n^^^^^\n", tweet.Retweeted, tweet.FullText)
						time.Sleep(1 * time.Second)
						dp.unRetweet(tweet.ID)
					} else {
						fmt.Printf("\n\t{%t} Timeline tweet: '%s'\n^^^^^\n",
							tweet.Retweeted, tweet.FullText)
						// Put a second between each deletion
						time.Sleep(1 * time.Second)
						dp.deleteTweet(tweet.ID)
					}
				}
			} else {
				fmt.Println("No more tweets. Exiting out of infinite loop")
				break
			}
		}
	}

}

func (dp *DelPost) unRetweet(id int64) error {
	t, _, err := dp.tClient.Statuses.Unretweet(id, dp.statusUnRTParams)
	if err != nil {
		fmt.Printf("\tGot an error deleting RT: %+v\n", err)
		return err
	}
	fmt.Printf("\n\tUnRT'd tweet: '%s'\n^ ^ ^ ^ ^\n", t.FullText)
	return nil
}

func (dp *DelPost) deleteTweet(id int64) error {
	t, _, err := dp.tClient.Statuses.Destroy(id, dp.destroyParams)
	if err != nil {
		fmt.Printf("\tGot an error deleting tweet: %+v\n", err)
		return err
	}
	fmt.Printf("\n\tDeleted Tweet: '%s'\n^^^^^\n", t.FullText)
	return nil
}

func (dp *DelPost) setUpTwitterClient() {
	config := oauth1.NewConfig(dp.cKey, dp.cSecret)
	token := oauth1.NewToken(dp.aToken, dp.aSecret)
	// OAuth1 http.Client will automatically authorize Requests
	httpClient := config.Client(oauth1.NoContext, token)
	// Twitter client
	dp.tClient = twitter.NewClient(httpClient)
}

func (dp *DelPost) readConfigFile(fname string) error {
	if fname == "" {
		log.Fatal("No configuration file name given. Exiting.")
	}
	file, err := os.Open(fname)
	if err != nil {
		fmt.Printf("Could not open pwfile '%s' for reading: %s\n", fname, err)
		log.Panic("Exiting")
	}
	if dp.configFile == "" {
		log.Fatal("No config file was configured")
	}

	// These are the lines we'll be looking for
	uLine := regexp.MustCompile(`user="?(?P<cKey>[^"]+)"?$`)
	ckLine := regexp.MustCompile(`consumer-key="?(?P<cKey>[^"]+)"?$`)
	csLine := regexp.MustCompile(`consumer-secret="?(?P<cSecret>[^"]+)"?$`)
	atLine := regexp.MustCompile(`access-token="?(?P<aToken>[^"]+)"?$`)
	asLine := regexp.MustCompile(`access-secret="?(?P<aSecret>[^"]+)"?$`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if uLine.MatchString(line) {
			matches := uLine.FindStringSubmatch(line)
			dp.user = matches[1]
		}
		if ckLine.MatchString(line) {
			matches := ckLine.FindStringSubmatch(line)
			dp.cKey = matches[1]
		}
		if csLine.MatchString(line) {
			matches := csLine.FindStringSubmatch(line)
			dp.cSecret = matches[1]
		}
		if atLine.MatchString(line) {
			matches := atLine.FindStringSubmatch(line)
			dp.aToken = matches[1]
		}
		if asLine.MatchString(line) {
			matches := asLine.FindStringSubmatch(line)
			dp.aSecret = matches[1]
		}
	}

	defer file.Close()

	// Make sure we got all the info we needed.
	if dp.user == "" || dp.cKey == "" || dp.cSecret == "" || dp.aToken == "" || dp.aSecret == "" {
		if dp.user == "" {
			log.Println("Missing username.")
		} else {
			log.Printf("Have username of '%s'\n", dp.user)
		}
		if dp.cKey == "" {
			log.Println("Missing Consumer Key.")
		} else {
			log.Printf("Have consumer key of '%s...'\n", dp.cKey[:4])
		}
		if dp.cSecret == "" {
			log.Println("Missing Consumer Secret.")
		} else {
			log.Printf("Have consumer secret of '%s...'\n", dp.cSecret[:4])
		}
		if dp.aToken == "" {
			log.Println("Missing Application Token.")
		} else {
			log.Printf("Have app token of '%s...'\n", dp.aToken[:4])
		}
		if dp.aSecret == "" {
			log.Println("Missing Application Secret")
		} else {
			log.Printf("Have app secret of '%s...'\n", dp.aSecret[:4])
		}
		log.Panic("Cannot continue. Exiting.")
	}
	// Don't Panic!
	return nil
}
