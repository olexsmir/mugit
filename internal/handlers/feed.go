package handlers

import (
	"net/http"
	"net/url"

	"github.com/gorilla/feeds"
)

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

	feed := &feeds.Feed{
		Title:       repoName,
		Link:        &feeds.Link{Href: feedLink},
		Description: desc,
	}

	// branches
	branches, err := repo.Branches()
	if err != nil {
		h.write500(w, err)
		return
	}

	for _, branch := range branches {
		href, _ := url.JoinPath("http://", h.c.Meta.Host, repoName, "tree", branch.Name)
		feed.Items = append(feed.Items, &feeds.Item{
			Id:      "b:" + branch.Name,
			Title:   "branch: " + branch.Name,
			Link:    &feeds.Link{Href: href},
			Updated: branch.LastUpdate,
		})
	}

	// tags
	tags, err := repo.Tags()
	if err != nil {
		h.write500(w, err)
	}

	for _, tag := range tags {
		href, _ := url.JoinPath("http://", h.c.Meta.Host, repoName, "tree", tag.Name())
		feed.Items = append(feed.Items, &feeds.Item{
			Id:      "t:" + tag.Name(),
			Title:   "tag: " + tag.Name(),
			Link:    &feeds.Link{Href: href},
			Updated: tag.When(),
			Content: tag.Message(),
		})
	}

	rss, err := feed.ToRss()
	if err != nil {
		h.write500(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/rss+xml")
	w.Write([]byte(rss))
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

	feed := &feeds.Feed{
		Title:       h.c.Meta.Host,
		Link:        &feeds.Link{Href: feedLink},
		Description: h.c.Meta.Description,
	}

	for _, repo := range repos {
		href, _ := url.JoinPath("http://", h.c.Meta.Host, repo.Name)
		feed.Items = append(feed.Items, &feeds.Item{
			Title:       repo.Name,
			Link:        &feeds.Link{Href: href},
			Description: repo.Desc,
			Id:          repo.Name,
			Updated:     repo.LastCommit,
			Content:     repo.Desc,
		})
	}

	rss, err := feed.ToRss()
	if err != nil {
		h.write500(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/rss+xml")
	w.Write([]byte(rss))
}
