module github.com/PichuChen/postgresql-gobbs

go 1.24.2

require (
	github.com/Ptt-official-app/go-bbs v0.12.0
	github.com/lib/pq v1.10.9
)

require golang.org/x/text v0.3.8 // indirect

replace github.com/Ptt-official-app/go-bbs => ../go-bbs
