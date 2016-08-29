package command

import (
	"flag"
	"fmt"
	"github.com/mitchellh/cli"
	"github.com/sourcegraph/go-papertrail/papertrail"
	"regexp"
	"strings"
	"time"
)

type SearchCommand struct {
	Ui cli.ColoredUi
}

func (c *SearchCommand) Help() string {
	return "pt search {args}"
}

func (c *SearchCommand) Synopsis() string {
	return "search for query in papertrail"
}

func (c *SearchCommand) Run(args []string) int {
	var (
		follow   bool
		kv       bool
		systemID string
		groupID  string
	)

	cmdFlags := flag.NewFlagSet("search", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }
	cmdFlags.BoolVar(&follow, "follow", true, "follow")
	cmdFlags.BoolVar(&kv, "kv", true, "kv")
	cmdFlags.StringVar(&systemID, "system", "", "system")
	cmdFlags.StringVar(&groupID, "group", "", "group ID (filter)")

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	token, err := papertrail.ReadToken()
	if err == papertrail.ErrNoTokenFound {
		c.Ui.Error("No Papertrail API token found; exiting.\n\npapertrail-go requires a valid Papertrail API token (which you can obtain from https://papertrailapp.com/user/edit) to be set in the PAPERTRAIL_API_TOKEN environment variable or in ~/.papertrail.yml (in the format `token: MYTOKEN`).")
		return 1
	} else if err != nil {
		c.Ui.Error(fmt.Sprintf("%v", err))
		return 1
	}

	client := papertrail.NewClient((&papertrail.TokenTransport{Token: token}).Client())

	queryTerms := cmdFlags.Args()
	if len(queryTerms) == 0 {
		c.Ui.Warn("You need at least one query term")
		return 0
	}

	opt := papertrail.SearchOptions{
		SystemID: systemID,
		GroupID:  groupID,
		Query:    strings.Join(queryTerms, " "),
	}

	minTimeAgo := 0 * time.Second
	delay := 2 * time.Second

	if minTimeAgo == 0 {
		opt.MinTime = time.Now().In(time.UTC).Add(-48 * time.Hour)
	}

	stopWhenEmpty := !follow && (minTimeAgo == 0)
	polling := false

	// seen := make(map[string]bool)
	for {
		searchResp, _, err := client.Search(opt)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("%v", err))
		}

		if len(searchResp.Events) == 0 {
			if stopWhenEmpty {
				return 0
			} else {
				// No more messages are immediately available, so now we'll just
				// poll periodically.
				polling = true
			}
		}
		for _, e := range searchResp.Events {
			// var prog string
			// if e.Program != nil {
			//  prog = *e.Program
			// }
			// e.ReceivedAt, e.SourceName, e.Facility, prog, e.Message

			if kv {
				parts := strings.Split(e.Message, " ")
				start := 0
				for i, part := range parts {
					if !strings.Contains(part, "=") {
						continue
					}
					start = i
					break
				}

				re := regexp.MustCompile(`\[(.*?)\]`)
				matches := re.FindAllString(e.Message, -1)

				ts := ""
				if len(matches) > 0 {
					ts = matches[0]
				}
				c.Ui.Output(fmt.Sprintf("ts=%s %s", ts, strings.Join(parts[start:], "\t")))
			} else {
				c.Ui.Output(e.Message)
			}
		}

		opt.MinID = searchResp.MaxID

		if polling {
			time.Sleep(delay)
		}
	}
}
