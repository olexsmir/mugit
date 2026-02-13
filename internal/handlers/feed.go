package handlers

import (
	"net/http"

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
	feed := &feeds.Feed{
		Title:       repoName,
		Link:        &feeds.Link{Href: h.c.Meta.Host + "/" + repoName},
		Description: desc,
	}

	// branches
	branches, err := repo.Branches()
	if err != nil {
		h.write500(w, err)
		return
	}

	for _, branch := range branches {
		feed.Items = append(feed.Items, &feeds.Item{
			Id:      "b:" + branch.Name,
			Title:   "branch: " + branch.Name,
			Link:    &feeds.Link{Href: h.c.Meta.Host + "/tree/" + branch.Name},
			Updated: branch.LastUpdate,
		})
	}

	// tags
	tags, err := repo.Tags()
	if err != nil {
		h.write500(w, err)
	}

	for _, tag := range tags {
		feed.Items = append(feed.Items, &feeds.Item{
			Id:          "t:" + tag.Name(),
			Title:       "tag: " + tag.Name(),
			Link:        &feeds.Link{Href: h.c.Meta.Host + "/tree/" + tag.Name()},
			Description: desc,
			Updated:     tag.When(),
			Content:     tag.Message(),
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

	feed := &feeds.Feed{
		Title:       h.c.Meta.Host,
		Link:        &feeds.Link{Href: h.c.Meta.Host},
		Description: h.c.Meta.Description,
	}

	for _, repo := range repos {
		feed.Items = append(feed.Items, &feeds.Item{
			Title:       repo.Name,
			Link:        &feeds.Link{Href: h.c.Meta.Host + "/" + repo.Name},
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
