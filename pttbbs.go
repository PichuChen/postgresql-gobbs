package pttbbs

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"database/sql"

	"github.com/Ptt-official-app/go-bbs"
	_ "github.com/lib/pq"
)

type Connector struct {
	bbs.UnimplementedConnector
	home string
	db   *sql.DB
}

func init() {
	bbs.Register("postgresql", &Connector{})
}

// Open connect a file directory or SHMs, dataSourceName pointer to bbs home
// And it can append argument for SHM
// for example `file:///home/bbs/?UTMP=1993`
func (c *Connector) Open(dataSourceName string) error {
	slog.Info("Open", "dataSourceName", dataSourceName)
	var err error

	// Connect to the PostgreSQL database
	// connStr := "user=pttap2 pass=f1808ebbe94e87fa8080d47fadf955f296cf3236 dbname=pttap sslmode=verify-ca"
	c.db, err = sql.Open("postgres", dataSourceName)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Check if the connection is alive
	if err := c.db.Ping(); err != nil {
		slog.Error("Failed to ping database", "error", err)
		return fmt.Errorf("failed to ping database: %w", err)
	}
	slog.Info("Connected to database successfully")
	c.home = dataSourceName
	return nil
}

func (c *Connector) GetUserRecordsPath() (string, error) {
	return "/users", nil
}

type UserRecord struct {
	bbs.UnimplementedUserRecord
}

func (u *UserRecord) UserID() string         { return "pichu" }
func (u *UserRecord) HashedPassword() string { return "" }
func (u *UserRecord) VerifyPassword(password string) error {
	return nil
}
func (u *UserRecord) Nickname() string     { return "" }
func (u *UserRecord) RealName() string     { return "" }
func (u *UserRecord) NumLoginDays() int    { return 0 }
func (u *UserRecord) NumPosts() int        { return 0 }
func (u *UserRecord) Money() int           { return 0 }
func (u *UserRecord) LastLogin() time.Time { return time.Time{} }
func (u *UserRecord) LastHost() string     { return "" }
func (u *UserRecord) UserFlag() uint32     { return 0 }

func (c *Connector) ReadUserRecordsFile(path string) ([]bbs.UserRecord, error) {
	mockedUserList := []bbs.UserRecord{&UserRecord{}}
	return mockedUserList, nil

}

func (c *Connector) GetBoardRecordsPath() (string, error) {
	return "/boards", nil
}

type BoardRecord struct {
	bbs.UnimplementedBoardRecord
	name  string
	bid   int // it should be index of original .BRD file
	extra map[string]interface{}
}

func (b *BoardRecord) Title() string {
	return b.extra["Title"].(string)
}
func (b *BoardRecord) BoardID() string {
	return b.name
}

func (c *Connector) ReadBoardRecordsFile(path string) ([]bbs.BoardRecord, error) {
	stmt := `SELECT name, bid, extra FROM boards`
	rows, err := c.db.Query(stmt)
	if err != nil {
		slog.Error("Failed to query database", "error", err)
		return nil, fmt.Errorf("failed to query database: %w", err)
	}

	defer rows.Close()
	var boardRecords []bbs.BoardRecord
	for rows.Next() {
		r := &BoardRecord{}
		var name string
		var bid int
		var extra string
		if err := rows.Scan(&name, &bid, &extra); err != nil {
			slog.Error("Failed to scan row", "error", err)
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		r.name = name
		r.bid = bid
		r.extra = make(map[string]interface{})
		if err := json.Unmarshal([]byte(extra), &r.extra); err != nil {
			slog.Error("Failed to unmarshal extra", "error", err)
			return nil, fmt.Errorf("failed to unmarshal extra: %w", err)
		}
		boardRecords = append(boardRecords, r)
	}
	if err := rows.Err(); err != nil {
		slog.Error("Error occurred during row iteration", "error", err)
		return nil, fmt.Errorf("error occurred during row iteration: %w", err)
	}
	// If no records found, return an empty slice
	if len(boardRecords) == 0 {
		slog.Info("No board records found")
		return []bbs.BoardRecord{}, nil
	}
	// If records found, return the list
	slog.Info("Found board records", "count", len(boardRecords))
	// Return the list of board records
	return boardRecords, nil
}

type ArticleRecord struct {
	bbs.UnimplementedArticleRecord
	board string
	aid   string
	extra map[string]interface{}
	// Filename string
	// BoardID  string
	// Title    string
	// Author   string
	// Ctime    time.Time
}

func (u *ArticleRecord) Filename() string                  { return u.aid }
func (u *ArticleRecord) Modified() time.Time               { return time.Time{} }
func (u *ArticleRecord) SetModified(newModified time.Time) {}
func (u *ArticleRecord) Recommend() int                    { return u.extra["pushcount"].(int) }
func (u *ArticleRecord) Date() string                      { return u.extra["list_date"].(string) }
func (u *ArticleRecord) Title() string                     { return u.extra["title"].(string) }
func (u *ArticleRecord) Money() int                        { return u.extra["money"].(int) }
func (u *ArticleRecord) Owner() string                     { return u.extra["author"].(string) }

func (c *Connector) GetBoardArticleRecordsPath(boardID string) (string, error) {
	slog.Info("GetBoardArticleRecordsPath", "boardID", boardID)
	return fmt.Sprintf("%s", boardID), nil
}

func (c *Connector) ReadArticleRecordsFile(filename string, offset, length uint) ([]bbs.ArticleRecord, error) {
	return c.ReadArticleRecordsFileFromArticleTable(filename, offset, length)
}

func (c *Connector) ReadArticleRecordsFileFromArticleTable(boardID string, offset, length uint) ([]bbs.ArticleRecord, error) {
	slog.Info("ReadArticleRecordsFileFromArticleTable", "boardID", boardID, "offset", offset, "length", length)
	if length > 1000000 {
		slog.Warn("Length exceeds 1000000, limiting to 1000000")
		length = 1000000
	}
	stmt := `SELECT id, title, board, index, author, date, pushcount, mark, url, is_deleted, extra
        FROM article WHERE UPPER(board) = UPPER($1) ORDER BY index ASC LIMIT $2 OFFSET $3`
	rows, err := c.db.Query(stmt, boardID, length, offset)
	if err != nil {
		slog.Error("Failed to query database", "error", err)
		return nil, fmt.Errorf("failed to query database: %w", err)
	}
	defer rows.Close()

	var articleRecords []bbs.ArticleRecord
	for rows.Next() {
		r := ArticleRecord{}
		var id, title, board, mark, url, author, date string
		var idx, pushcount int
		var isDeleted bool
		var extraStr string

		if err := rows.Scan(&id, &title, &board, &idx, &author, &date, &pushcount, &mark, &url, &isDeleted, &extraStr); err != nil {
			slog.Error("Failed to scan row", "error", err)
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		r.board = board
		r.aid = Aidu2Aidc(Fn2Aidu("M." + id))
		r.extra = make(map[string]interface{})
		// 填入主要欄位到 extra
		r.extra["title"] = title
		r.extra["author"] = author
		r.extra["list_date"] = date
		r.extra["pushcount"] = pushcount
		r.extra["mark"] = mark
		r.extra["url"] = url
		r.extra["is_deleted"] = isDeleted
		r.extra["index"] = idx
		slog.Info("ReadArticleRecordsFileFromArticleTable", "board", board, "aid", r.aid, "title", title, "author", author, "date", date, "pushcount", pushcount, "mark", mark, "url", url, "is_deleted", isDeleted)

		// 合併 extra JSONB 欄位
		if extraStr != "" {
			var extraMap map[string]interface{}
			if err := json.Unmarshal([]byte(extraStr), &extraMap); err != nil {
				slog.Error("Failed to unmarshal extra", "error", err)
				return nil, fmt.Errorf("failed to unmarshal extra: %w", err)
			}
			for k, v := range extraMap {
				r.extra[k] = v
			}
		}

		articleRecords = append(articleRecords, &r)
	}
	if err := rows.Err(); err != nil {
		slog.Error("Error occurred during row iteration", "error", err)
		return nil, fmt.Errorf("error occurred during row iteration: %w", err)
	}
	return articleRecords, nil
}

func (c *Connector) ReadArticleRecordsFileFromPostsTable(filename string, offset, length uint) ([]bbs.ArticleRecord, error) {
	slog.Info("ReadArticleRecordsFile", "filename", filename)
	stmt := `SELECT board, aid, extra FROM posts WHERE UPPER(board) = UPPER($1) ORDER BY aid ASC`
	rows, err := c.db.Query(stmt, filename)
	if err != nil {
		slog.Error("Failed to query database", "error", err)
		return nil, fmt.Errorf("failed to query database: %w", err)
	}
	defer rows.Close()

	var articleRecords []bbs.ArticleRecord
	for rows.Next() {
		r := ArticleRecord{}
		var board string
		var aid string
		var extra string
		if err := rows.Scan(&board, &aid, &extra); err != nil {
			slog.Error("Failed to scan row", "error", err)
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		r.board = board
		r.aid = aid
		r.extra = make(map[string]interface{})
		if err := json.Unmarshal([]byte(extra), &r.extra); err != nil {
			slog.Error("Failed to unmarshal extra", "error", err)
			return nil, fmt.Errorf("failed to unmarshal extra: %w", err)
		}

		articleRecords = append(articleRecords, &r)
	}
	if err := rows.Err(); err != nil {
		slog.Error("Error occurred during row iteration", "error", err)
		return nil, fmt.Errorf("error occurred during row iteration: %w", err)
	}
	return articleRecords, nil
}
