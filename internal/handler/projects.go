package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/jmstewart1127/openlogs/internal/db"
)

// projectsView is the data for the project list page.
type projectsView struct {
	Projects []db.Project
}

// projectSettingsView is the data for a single project's settings page.
type projectSettingsView struct {
	Project *db.Project
}

// Projects renders the list of all projects.
func (a *App) Projects(w http.ResponseWriter, r *http.Request) {
	projects, err := a.DB.ListProjects()
	if err != nil {
		log.Printf("projects: list failed: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	a.render(w, r, "projects", projectsView{Projects: projects}, "")
}

// CreateProject creates a new project and redirects to its settings page, where
// the freshly generated API key is displayed.
func (a *App) CreateProject(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	project, err := a.DB.CreateProject(name)
	if err != nil {
		if errors.Is(err, db.ErrDuplicateName) || name == "" {
			projects, lerr := a.DB.ListProjects()
			if lerr != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			flash := "A project with that name already exists"
			if name == "" {
				flash = "Project name must not be empty"
			}
			w.WriteHeader(http.StatusBadRequest)
			a.render(w, r, "projects", projectsView{Projects: projects}, flash)
			return
		}
		log.Printf("projects: create failed: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/projects/"+project.ID+"/settings", http.StatusSeeOther)
}

// ProjectSettings renders a project's settings page.
func (a *App) ProjectSettings(w http.ResponseWriter, r *http.Request) {
	project := a.loadProject(w, r)
	if project == nil {
		return
	}
	a.render(w, r, "project-settings", projectSettingsView{Project: project}, "")
}

// RenameProject updates a project's name.
func (a *App) RenameProject(w http.ResponseWriter, r *http.Request) {
	project := a.loadProject(w, r)
	if project == nil {
		return
	}
	err := a.DB.RenameProject(project.ID, r.FormValue("name"))
	if err != nil {
		flash := "Could not rename project"
		if errors.Is(err, db.ErrDuplicateName) {
			flash = "A project with that name already exists"
		}
		a.render(w, r, "project-settings", projectSettingsView{Project: project}, flash)
		return
	}
	http.Redirect(w, r, "/projects/"+project.ID+"/settings", http.StatusSeeOther)
}

// RegenerateKey issues a new API key, invalidating the old one.
func (a *App) RegenerateKey(w http.ResponseWriter, r *http.Request) {
	project := a.loadProject(w, r)
	if project == nil {
		return
	}
	if _, err := a.DB.RegenerateAPIKey(project.ID); err != nil {
		log.Printf("projects: regenerate key failed: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/projects/"+project.ID+"/settings", http.StatusSeeOther)
}

// DeleteProject removes a project and all its logs.
func (a *App) DeleteProject(w http.ResponseWriter, r *http.Request) {
	project := a.loadProject(w, r)
	if project == nil {
		return
	}
	if err := a.DB.DeleteProject(project.ID); err != nil {
		log.Printf("projects: delete failed: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/projects", http.StatusSeeOther)
}

// loadProject fetches the project named in the {id} path value, writing a 404 and
// returning nil if it does not exist.
func (a *App) loadProject(w http.ResponseWriter, r *http.Request) *db.Project {
	id := r.PathValue("id")
	project, err := a.DB.GetProjectByID(id)
	if err != nil {
		log.Printf("projects: lookup failed: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return nil
	}
	if project == nil {
		http.NotFound(w, r)
		return nil
	}
	return project
}
