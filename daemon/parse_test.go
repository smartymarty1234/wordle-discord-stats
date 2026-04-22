package daemon

import (
	"reflect"
	"testing"
)

func TestParseAggregateScores(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []userScore
	}{
		{
			name: "discord ID mentions only",
			content: `**Your group is on a 261 day streak!** Here are yesterday's results:
👑 4/6: <@454692930943320065>
5/6: <@382714563101196290>
X/6: <@111111111111111111>`,
			want: []userScore{
				{userID: "454692930943320065", score: 4, complete: true},
				{userID: "382714563101196290", score: 5, complete: true},
				{userID: "111111111111111111", score: 0, complete: false},
			},
		},
		{
			name: "plain @name only",
			content: `Here are yesterday's results:
X/6: @chimchi cruncher`,
			want: []userScore{
				{fixedNick: "chimchi cruncher", score: 0, complete: false},
			},
		},
		{
			name: "mixed plain and Discord ID on same line",
			content: `Here are yesterday's results:
👑 3/6: <@222001869906640896>
4/6: @rust cruncher 2 <@454692930943320065> <@541689076126842921>
5/6: <@328236081315053569>`,
			want: []userScore{
				{userID: "222001869906640896", score: 3, complete: true},
				{fixedNick: "rust cruncher 2", score: 4, complete: true},
				{userID: "454692930943320065", score: 4, complete: true},
				{userID: "541689076126842921", score: 4, complete: true},
				{userID: "328236081315053569", score: 5, complete: true},
			},
		},
		{
			name: "multiple Discord IDs on same line",
			content: `Here are yesterday's results:
4/6: <@382714563101196290> <@541689076126842921>`,
			want: []userScore{
				{userID: "382714563101196290", score: 4, complete: true},
				{userID: "541689076126842921", score: 4, complete: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAggregateScores(tt.content)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got  %+v\nwant %+v", got, tt.want)
			}
		})
	}
}
