package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Steam API types

type WishlistResponse struct {
	Response struct {
		Items []WishlistItem `json:"items"`
	} `json:"response"`
}

type WishlistItem struct {
	AppID    int `json:"appid"`
	Priority int `json:"priority"`
}

type AppDetailsResponse map[string]struct {
	Success bool `json:"success"`
	Data    struct {
		Name string `json:"name"`
	} `json:"data"`
}

// OPML types

type OPML struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Head    OPMLHead `xml:"head"`
	Body    OPMLBody `xml:"body"`
}

type OPMLHead struct {
	Title       string `xml:"title"`
	DateCreated string `xml:"dateCreated"`
}

type OPMLBody struct {
	Outlines []OPMLOutline `xml:"outline"`
}

type OPMLOutline struct {
	Text    string `xml:"text,attr"`
	Title   string `xml:"title,attr"`
	Type    string `xml:"type,attr"`
	XMLURL  string `xml:"xmlUrl,attr"`
	HTMLURL string `xml:"htmlUrl,attr"`
}

var (
	httpClient = &http.Client{Timeout: 15 * time.Second}

	steamIDRegex = regexp.MustCompile(`steamid["\\/]*:\s*["\\/]*(\d{17})`)
	validUserID  = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

	// Caches: Steam IDs and app names never change, so no TTL needed.
	steamIDCache   = make(map[string]string) // vanity URL -> numeric Steam ID
	steamIDCacheMu sync.RWMutex

	appNameCache   = make(map[int]string) // appID -> game name
	appNameCacheMu sync.RWMutex
)

// resolveSteamID extracts the numeric Steam ID from the wishlist page for a vanity URL.
func resolveSteamID(userID string) (string, error) {
	steamIDCacheMu.RLock()
	if id, ok := steamIDCache[userID]; ok {
		steamIDCacheMu.RUnlock()
		return id, nil
	}
	steamIDCacheMu.RUnlock()

	url := fmt.Sprintf("https://store.steampowered.com/wishlist/id/%s/", userID)
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetching wishlist page: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading wishlist page: %w", err)
	}

	matches := steamIDRegex.FindAllStringSubmatch(string(body), -1)
	for _, m := range matches {
		steamIDCacheMu.Lock()
		steamIDCache[userID] = m[1]
		steamIDCacheMu.Unlock()
		return m[1], nil
	}
	return "", fmt.Errorf("could not find Steam ID for user %q", userID)
}

func fetchWishlist(steamID string) ([]WishlistItem, error) {
	url := fmt.Sprintf("https://api.steampowered.com/IWishlistService/GetWishlist/v1/?steamid=%s", steamID)
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching wishlist: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steam API returned status %d", resp.StatusCode)
	}

	var result WishlistResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding wishlist: %w", err)
	}

	if len(result.Response.Items) == 0 {
		return nil, fmt.Errorf("no games found in wishlist")
	}

	return result.Response.Items, nil
}

func fetchAppName(appID int) string {
	appNameCacheMu.RLock()
	if name, ok := appNameCache[appID]; ok {
		appNameCacheMu.RUnlock()
		return name
	}
	appNameCacheMu.RUnlock()

	url := fmt.Sprintf("https://store.steampowered.com/api/appdetails?appids=%d&filters=basic", appID)
	resp, err := httpClient.Get(url)
	if err != nil {
		return strconv.Itoa(appID)
	}
	defer resp.Body.Close()

	var details AppDetailsResponse
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return strconv.Itoa(appID)
	}

	key := strconv.Itoa(appID)
	if d, ok := details[key]; ok && d.Success && d.Data.Name != "" {
		appNameCacheMu.Lock()
		appNameCache[appID] = d.Data.Name
		appNameCacheMu.Unlock()
		return d.Data.Name
	}
	return key
}

type gameInfo struct {
	AppID int
	Name  string
}

func fetchAllNames(items []WishlistItem) []gameInfo {
	games := make([]gameInfo, len(items))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10) // limit concurrency

	for i, item := range items {
		wg.Add(1)
		go func(i int, appID int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			games[i] = gameInfo{AppID: appID, Name: fetchAppName(appID)}
		}(i, item.AppID)
	}

	wg.Wait()
	return games
}

func buildOPML(userID string, games []gameInfo) OPML {
	sort.Slice(games, func(i, j int) bool {
		return strings.ToLower(games[i].Name) < strings.ToLower(games[j].Name)
	})

	outlines := make([]OPMLOutline, 0, len(games))
	for _, g := range games {
		outlines = append(outlines, OPMLOutline{
			Text:    g.Name,
			Title:   g.Name,
			Type:    "rss",
			XMLURL:  fmt.Sprintf("https://store.steampowered.com/feeds/news/app/%d/", g.AppID),
			HTMLURL: fmt.Sprintf("https://store.steampowered.com/app/%d/", g.AppID),
		})
	}

	return OPML{
		Version: "2.0",
		Head: OPMLHead{
			Title:       fmt.Sprintf("Steam Wishlist RSS - %s", userID),
			DateCreated: time.Now().UTC().Format(time.RFC1123Z),
		},
		Body: OPMLBody{Outlines: outlines},
	}
}

func handleWishlist(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimPrefix(r.URL.Path, "/wishlist/")
	userID = strings.TrimSuffix(userID, "/")

	if userID == "" || !validUserID.MatchString(userID) {
		http.Error(w, "Usage: /wishlist/{steam_user_id}", http.StatusBadRequest)
		return
	}

	log.Printf("Request for user: %s", userID)

	steamID, err := resolveSteamID(userID)
	if err != nil {
		log.Printf("Error resolving Steam ID for %s: %v", userID, err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	log.Printf("Resolved %s -> Steam ID %s", userID, steamID)

	items, err := fetchWishlist(steamID)
	if err != nil {
		log.Printf("Error fetching wishlist: %v", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	log.Printf("Found %d games in wishlist", len(items))

	games := fetchAllNames(items)

	appNameCacheMu.RLock()
	cachedNames := len(appNameCache)
	appNameCacheMu.RUnlock()
	log.Printf("App name cache size: %d", cachedNames)

	opml := buildOPML(userID, games)

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-wishlist.opml"`, userID))

	fmt.Fprint(w, xml.Header)
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(opml); err != nil {
		log.Printf("Error encoding OPML: %v", err)
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	http.HandleFunc("/wishlist/", handleWishlist)

	log.Printf("Listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
