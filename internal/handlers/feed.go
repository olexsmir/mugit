package handlers

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"time"
)

type rssFeedXML struct {
	XMLName xml.Name      `xml:"rss"`
	Version string        `xml:"version,attr"`
	Channel rssChannelXML `xml:"channel"`
}

type rssChannelXML struct {
	Title       string       `xml:"title"`
	Link        string       `xml:"link"`
	Description string       `xml:"description"`
	Items       []rssItemXML `xml:"item"`
}

type rssItemXML struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Guid        string `xml:"guid"`
	Description string `xml:"description,omitempty"`
	PubDate     string `xml:"pubDate,omitempty"`
}

func (h *handlers) repoFeedHandler(w http.ResponseWriter, r *http.Request) {
	repo, err := h.openPublicRepo(r.PathValue("name"), "")
	if err != nil {
		h.write404(w, err)
		return
	}

	desc, err := repo.Description()
	if err != nil {
		h.write500(w, err)
		return
	}

	repoName := repo.Name()
	feedLink, err := url.JoinPath("http://", h.c.Meta.Host, repoName)
	if err != nil {
		h.write500(w, err)
		return
	}

	feed := rssFeedXML{
		Version: "2.0",
		Channel: rssChannelXML{
			Title:       repoName,
			Link:        feedLink,
			Description: desc,
		},
	}

	// branches
	branches, err := repo.Branches()
	if err != nil {
		h.write500(w, err)
		return
	}

	for _, branch := range branches {
		href, _ := url.JoinPath("http://", h.c.Meta.Host, repoName, "tree", branch.Name)
		it := rssItemXML{
			Title: "branch: " + branch.Name,
			Link:  href,
			Guid:  href,
		}
		if !branch.LastUpdate.IsZero() {
			it.PubDate = branch.LastUpdate.Format(time.RFC1123Z)
		}
		feed.Channel.Items = append(feed.Channel.Items, it)
	}

	// tags
	tags, err := repo.Tags()
	if err == nil {
		for _, tag := range tags {
			href, _ := url.JoinPath("http://", h.c.Meta.Host, repoName, "tree", tag.Name())
			it := rssItemXML{
				Title:       "tag: " + tag.Name(),
				Link:        href,
				Guid:        href,
				Description: tag.Message(),
			}
			if !tag.When().IsZero() {
				it.PubDate = tag.When().Format(time.RFC1123Z)
			}
			feed.Channel.Items = append(feed.Channel.Items, it)
		}
	}

	w.Header().Set("Content-Type", "application/rss+xml")
	w.Write([]byte(xml.Header))
	if err := xml.NewEncoder(w).Encode(feed); err != nil {
		h.write500(w, err)
		return
	}
}

func (h *handlers) indexFeedHandler(w http.ResponseWriter, r *http.Request) {
	repos, err := h.listPublicRepos()
	if err != nil {
		h.write500(w, err)
		return
	}

	feedLink, err := url.JoinPath("http://", h.c.Meta.Host)
	if err != nil {
		h.write500(w, err)
		return
	}

	feed := rssFeedXML{
		Version: "2.0",
		Channel: rssChannelXML{
			Title:       h.c.Meta.Host,
			Link:        feedLink,
			Description: h.c.Meta.Description,
		},
	}

	for _, repo := range repos {
		href, _ := url.JoinPath("http://", h.c.Meta.Host, repo.Name)
		it := rssItemXML{
			Title:       repo.Name,
			Link:        href,
			Guid:        href,
			Description: repo.Desc,
		}
		if !repo.LastCommit.IsZero() {
			it.PubDate = repo.LastCommit.Format(time.RFC1123Z)
		}
		feed.Channel.Items = append(feed.Channel.Items, it)
	}

	w.Header().Set("Content-Type", "application/rss+xml")
	w.Write([]byte(xml.Header))
	if err := xml.NewEncoder(w).Encode(feed); err != nil {
		h.write500(w, err)
		return
	}
}
