package sonarqube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/exp/slices"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Portfolio used in Portfolio
type Portfolio struct {
	Key           string   `json:"key"`
	Name          string   `json:"name"`
	Desc          string   `json:"desc"`
	Qualifier     string   `json:"qualifier"`
	Visibility    string   `json:"visibility"`
	SelectionMode string   `json:"selectionMode"`
	Branch        string   `json:"branch"`
	Tags          []string `json:"tags"`
	Regexp        string   `json:"regexp"`
}

// Returns the resource represented by this file.
func resourceSonarqubePortfolio() *schema.Resource {
	return &schema.Resource{
		Create: resourceSonarqubePortfolioCreate,
		Read:   resourceSonarqubePortfolioRead,
		Update: resourceSonarqubePortfolioUpdate,
		Delete: resourceSonarqubePortfolioDelete,
		Importer: &schema.ResourceImporter{
			State: resourceSonarqubePortfolioImport,
		},

		// Define the fields of this schema.
		Schema: map[string]*schema.Schema{
			"key": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"description": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"qualifier": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"visibility": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "public",
				ForceNew: true, // I cant find an API endpoint for changing this, even though it's possible in the UI
				ValidateFunc: func(val any, key string) (warns []string, errs []error) {
					visibility := val.(string)
					validOptions := []string{"public", "private"}
					if !slices.Contains(validOptions, visibility) {
						errs = append(errs, fmt.Errorf("Accepted values are public or private for key %q, got: %s", key, val))
					}

					return
				},
			},
			"selection_mode": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "NONE",
				ForceNew: false,
				ValidateFunc: func(val any, key string) (warns []string, errs []error) {
					selectionMode := val.(string)
					validOptions := []string{"NONE", "MANUAL", "TAGS", "REGEXP", "REST"}
					if !slices.Contains(validOptions, selectionMode) {
						errs = append(errs, fmt.Errorf("Accepted values are NONE, MANUAL, TAGS, REGEXP or REST for key %q, got: %s", key, val))
					}
					return
				},
			},
			"branch": { // Only active for TAGS, REGEXP and REST
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"tags": { // Only active for TAGS
				Type:          schema.TypeList,
				Optional:      true,
				ForceNew:      false,
				ConflictsWith: []string{"regexp"},
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"regexp": { // Only active for REGEXP
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      false,
				ConflictsWith: []string{"tags"},
			},

			// TODO: MANUAL
			// "selectedProjects": [],
			// "projects": [],
		},
	}
}

func portfolioSetSelectionMode(d *schema.ResourceData, m interface{}, sonarQubeURL url.URL) error {
	var endpoint string
	switch selectionMode := d.Get("selection_mode"); selectionMode {
	case "NONE":
		endpoint = "/api/views/set_none_mode"
		sonarQubeURL.RawQuery = url.Values{
			"portfolio": []string{d.Get("key").(string)},
		}.Encode()

	case "MANUAL":
		endpoint = "/api/views/set_manual_mode"
		sonarQubeURL.RawQuery = url.Values{
			"portfolio": []string{d.Get("key").(string)},
		}.Encode()

	case "TAGS":
		if !d.HasChanges("branch", "tags") {
			return nil
		}

		endpoint = "/api/views/set_tags_mode"
		
		var tags []string
		for _, v := range d.Get("tags").([]interface{}) {
			tags = append(tags, fmt.Sprint(v))
		}
		tagsCSV := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(tags)), ","), "[]")
		sonarQubeURL.RawQuery = url.Values{
			"branch":    []string{d.Get("branch").(string)},
			"portfolio": []string{d.Get("key").(string)},
			"tags":      []string{tagsCSV},
		}.Encode()

	case "REGEXP":
		if !d.HasChanges("branch", "regexp") {
			return nil
		}

		endpoint = "/api/views/set_regexp_mode"
		sonarQubeURL.RawQuery = url.Values{
			"branch":    []string{d.Get("branch").(string)},
			"portfolio": []string{d.Get("key").(string)},
			"regexp":    []string{d.Get("regexp").(string)},
		}.Encode()

	case "REST":
		if !d.HasChange("branch") {
			return nil
		}

		endpoint = "/api/views/set_remaining_projects_mode"
		sonarQubeURL.RawQuery = url.Values{
			"branch":    []string{d.Get("branch").(string)},
			"portfolio": []string{d.Get("key").(string)},
		}.Encode()

	default:
		return fmt.Errorf("resourceSonarqubePortfolioCreate: selection_mode needs to be set to one of NONE, MANUAL, TAGS, REGEXP, REST")
	}

	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + endpoint

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL.String(),
		http.StatusNoContent,
		"resourceSonarqubePortfolioCreate",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func resourceSonarqubePortfolioCreate(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/views/create"

	sonarQubeURL.RawQuery = url.Values{
		"description": []string{d.Get("description").(string)},
		"key":         []string{d.Get("key").(string)},
		"name":        []string{d.Get("name").(string)},
		"visibility":  []string{d.Get("visibility").(string)},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL.String(),
		http.StatusOK,
		"resourceSonarqubePortfolioCreate",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = portfolioSetSelectionMode(d, m, m.(*ProviderConfiguration).sonarQubeURL)
	if err != nil {
		return err
	}

	// Decode response into struct
	portfolioResponse := Portfolio{}
	err = json.NewDecoder(resp.Body).Decode(&portfolioResponse)
	if err != nil {
		return fmt.Errorf("resourceSonarqubePortfolioCreate: Failed to decode json into struct: %+v", err)
	}

	d.SetId(portfolioResponse.Key)
	return resourceSonarqubePortfolioRead(d, m)
}

func resourceSonarqubePortfolioRead(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/views/show"
	sonarQubeURL.RawQuery = url.Values{
		"key": []string{d.Id()},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL.String(),
		http.StatusOK,
		"resourceSonarqubePortfolioRead",
	)
	if err != nil {
		return fmt.Errorf("resourceSonarqubePortfolioRead: Failed to read portfolio %+v: %+v", *d, err)

		// return err
	}
	defer resp.Body.Close()

	// Decode response into struct
	portfolioReadResponse := Portfolio{}
	err = json.NewDecoder(resp.Body).Decode(&portfolioReadResponse)
	if err != nil {
		return fmt.Errorf("resourceSonarqubePortfolioRead: Failed to decode json into struct: %+v", err)
	}

	d.SetId(portfolioReadResponse.Key)
	d.Set("key", portfolioReadResponse.Key)
	d.Set("name", portfolioReadResponse.Name)
	d.Set("description", portfolioReadResponse.Desc)
	d.Set("qualifier", portfolioReadResponse.Qualifier)
	d.Set("visibility", portfolioReadResponse.Visibility)
	d.Set("selection_mode", portfolioReadResponse.SelectionMode)
	d.Set("branch", portfolioReadResponse.Branch)
	d.Set("tags", portfolioReadResponse.Tags)
	d.Set("regexp", portfolioReadResponse.Regexp)

	return nil
}

func resourceSonarqubePortfolioUpdate(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL

	if d.HasChanges("name", "description") {
		sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/views/update"
		sonarQubeURL.RawQuery = url.Values{
			"key":         []string{d.Id()},
			"description": []string{d.Get("description").(string)},
			"name":        []string{d.Get("name").(string)},
		}.Encode()

		resp, err := httpRequestHelper(
			m.(*ProviderConfiguration).httpClient,
			"POST",
			sonarQubeURL.String(),
			http.StatusOK,
			"resourceSonarqubePortfolioUpdate",
		)
		if err != nil {
			return fmt.Errorf("error updating Sonarqube Portfolio Name and Description: %+v", err)
		}
		defer resp.Body.Close()
	}

	if d.HasChanges("selection_mode", "branch", "tags", "regexp") {
		err := portfolioSetSelectionMode(d, m, sonarQubeURL)
		if err != nil {
			return fmt.Errorf("error updating Sonarqube selection mode: %+v", err)
		}
	}

	return resourceSonarqubePortfolioRead(d, m)
}

func resourceSonarqubePortfolioDelete(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/views/delete"
	sonarQubeURL.RawQuery = url.Values{
		"key": []string{d.Id()},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL.String(),
		http.StatusNoContent,
		"resourceSonarqubePortfolioDelete",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func resourceSonarqubePortfolioImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	if err := resourceSonarqubePortfolioRead(d, m); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}
