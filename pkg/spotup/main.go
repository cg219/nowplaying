package spotup

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/cg219/nowplaying/internal/database"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

type Entry struct {
    Timestamp string `json:"ts"`
    Track string `json:"master_metadata_track_name"`
    Artist string `json:"master_metadata_album_artist_name"`
    Album string `json:"master_metadata_album_album_name"`
}

func Run() {
    slog.Info("Starting Spotify Upload")
    if len(os.Args) == 1 {
        log.Fatal("please add path to spotify export")
    }

    pathtofile := os.Args[1]

    slog.Info("File to read", "file", pathtofile)
    slog.Info("Reading file into memory")

    cwd, _ := os.Getwd()
    data, err := os.ReadFile(filepath.Join(cwd, pathtofile))

    if err != nil {
        log.Fatalf("Error Reading File: %s\n", err)
    }

    slog.Info("Connecting to database")

    conn := fmt.Sprintf("%s?authToken=%s", os.Getenv("TURSO_URL"), os.Getenv("TURSO_TOKEN"))
    db, _ := sql.Open("libsql", conn)
    defer db.Close()

    storage := database.New(db)

    slog.Info("Decoding Json")


    var structured []Entry
    if err := json.Unmarshal(data, &structured); err != nil {
        log.Fatal("Error Unmarshalling")
    }

    slog.Info("Storing...", "database", db)

    for _, e := range structured {
        if e.Track == "" {
            continue
        }

        parsed, err := time.Parse(time.RFC3339, e.Timestamp)
        if err != nil {
            slog.Error("ERROR")
        } 
        slog.Info("Entry", "song", e.Track, "artist", e.Artist, "album", e.Album, "timestamp", parsed.Unix())

        err = storage.AddToHistory(context.Background(), database.AddToHistoryParams{
            ArtistName: e.Artist,
            TrackName: e.Track,
            AlbumName: sql.NullString{ String: e.Album, Valid: e.Album != "" },
            Timestamp: parsed.UnixMilli(),
        })

        if err != nil {
            slog.Error("Error storing", "err", err)
        }
    }

    slog.Info("Transferring to scrobbles...", "database", db)

    // storage.HistoryToScrobbles(context.Background())

    slog.Info("Completed")
}
