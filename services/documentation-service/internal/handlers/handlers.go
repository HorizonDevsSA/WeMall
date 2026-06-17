package handlers

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/wemall/services/documentation-service/internal/models"
)

type Handler struct {
	tmpl          *template.Template
	apiCategories []models.APICategory
}

func NewHandler(tmpl *template.Template, apiCategories []models.APICategory) *Handler {
	return &Handler{
		tmpl:          tmpl,
		apiCategories: apiCategories,
	}
}

func (h *Handler) getCategoryBySlug(slug string) (models.APICategory, bool) {
	for _, c := range h.apiCategories {
		if c.Slug == slug {
			return c, true
		}
	}
	return models.APICategory{}, false
}

func (h *Handler) getEndpointsCount() int {
	count := 0
	for _, c := range h.apiCategories {
		count += len(c.Endpoints)
	}
	return count
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	stats := models.SystemStats{
		TotalCategories: len(h.apiCategories),
		TotalEndpoints:  h.getEndpointsCount(),
		Microservices:   7,
		Protocols:       3, // GraphQL, HTTP, gRPC
		GatewayPort:     "8080",
		Version:         "v2.1.0",
	}

	isHX := r.Header.Get("HX-Request") == "true"

	data := map[string]interface{}{
		"Categories":  h.apiCategories,
		"Stats":       stats,
		"ActiveSlug":  "dashboard",
		"ContentOnly": isHX,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if isHX {
		err := h.tmpl.ExecuteTemplate(w, "dashboard.html", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		err := h.tmpl.ExecuteTemplate(w, "layout.html", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (h *Handler) APIDetail(w http.ResponseWriter, r *http.Request) {
	// Simple path parsing. Path is expected to be "/flows/{slug}" or "/apis/{slug}"
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 || parts[2] == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	slug := parts[2]

	category, found := h.getCategoryBySlug(slug)
	if !found {
		http.NotFound(w, r)
		return
	}

	isHX := r.Header.Get("HX-Request") == "true"

	data := map[string]interface{}{
		"Categories":  h.apiCategories,
		"Category":    category,
		"ActiveSlug":  category.Slug,
		"ContentOnly": isHX,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if isHX {
		err := h.tmpl.ExecuteTemplate(w, "api_detail.html", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		err := h.tmpl.ExecuteTemplate(w, "layout.html", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (h *Handler) legalPage(w http.ResponseWriter, r *http.Request, templateName, activeSlug string) {
	isHX := r.Header.Get("HX-Request") == "true"

	data := map[string]interface{}{
		"Categories":  h.apiCategories,
		"ActiveSlug":  activeSlug,
		"ContentOnly": isHX,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if isHX {
		err := h.tmpl.ExecuteTemplate(w, templateName, data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		err := h.tmpl.ExecuteTemplate(w, "layout.html", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (h *Handler) LegalIndex(w http.ResponseWriter, r *http.Request) {
	h.legalPage(w, r, "legal_index.html", "legal")
}

func (h *Handler) LegalTerms(w http.ResponseWriter, r *http.Request) {
	h.legalPage(w, r, "legal_terms.html", "legal-terms")
}

func (h *Handler) LegalSeller(w http.ResponseWriter, r *http.Request) {
	h.legalPage(w, r, "legal_seller.html", "legal-seller")
}

func (h *Handler) LegalRider(w http.ResponseWriter, r *http.Request) {
	h.legalPage(w, r, "legal_rider.html", "legal-rider")
}

func (h *Handler) LegalPrivacy(w http.ResponseWriter, r *http.Request) {
	h.legalPage(w, r, "legal_privacy.html", "legal-privacy")
}

func (h *Handler) LegalDisputes(w http.ResponseWriter, r *http.Request) {
	h.legalPage(w, r, "legal_disputes.html", "legal-disputes")
}

func (h *Handler) LegalEULA(w http.ResponseWriter, r *http.Request) {
	h.legalPage(w, r, "legal_eula.html", "legal-eula")
}
