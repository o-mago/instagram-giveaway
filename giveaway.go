package main

import (
	"fmt"
	"math/rand"
	"regexp"
	"slices"
	"time"
)

type giveaway map[string][]string

var regex = regexp.MustCompile("@[^ ]*")

func (g *giveaway) Random(totalWinners int, blockList ...string) (giveaway, error) {
	if g == nil || len(*g) == 0 {
		return nil, fmt.Errorf("empty giveaway contenders")
	}

	winners := giveaway{}

	selectedIndexes := map[int]bool{}

	for len(winners) < totalWinners {
		if len(selectedIndexes) >= len(*g) {
			break
		}

		randomIndex := rand.Intn(len(*g))

		for selectedIndexes[randomIndex] {
			randomIndex = rand.Intn(len(*g))
		}

		index := 0
		for userName, mentions := range *g {
			if index == randomIndex {
				if slices.Contains(blockList, userName) {
					break
				}

				winners[userName] = mentions
			}

			index++
		}
	}

	return winners, nil
}

type startGiveawayInput struct {
	userName      string
	postCode      string
	token         string
	totalMentions int
	totalWinners  int
	blockList     []string
	shouldFilter  bool
}

func (m *model) startGiveaway(input startGiveawayInput) {
	userID, err := getUserInfo(input.userName)
	if err != nil {
		panic(err)
	}

	postID := ""

	nextURL := ""

	for {
		posts, err := getPostsData(userID, input.token, nextURL)
		if err != nil {
			panic(err)
		}

		nextURL = posts.Paging.Next

		for _, post := range posts.Data {
			if post.ShortCode == input.postCode {
				postID = post.ID

				break
			}
		}

		if postID != "" {
			break
		}

		if posts.Paging.Next == "" {
			break
		}
	}

	var commentsFinal []commentsData

	nextURL = ""

	for {
		comments, err := getCommentsData(postID, input.token, nextURL)
		if err != nil {
			panic(err)
		}

		nextURL = comments.Paging.Next

		commentsFinal = append(commentsFinal, comments.Data...)

		if comments.Paging.Next == "" {
			break
		}
	}

	contenders := giveaway{}

	for _, comment := range commentsFinal {
		mentions := regex.FindAllString(comment.Text, -1)

		contenders[comment.Username] = append(contenders[comment.Username], mentions...)
	}

	finalList := giveaway{}
	for username, mentions := range contenders {
		nonRepeatedMentions := slices.Compact(mentions)

		if len(nonRepeatedMentions) < input.totalMentions {
			continue
		}

		finalList[username] = nonRepeatedMentions
	}

	for i := 0; i < 10; i++ {
		m.percent += 0.1

		time.Sleep(time.Second)
	}

	winners, err := finalList.Random(input.totalWinners)
	if err != nil {
		panic(err)
	}

	m.winners = winners
	m.finish = true
}
