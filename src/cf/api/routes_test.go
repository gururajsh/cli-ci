package api_test

import (
	"cf"
	. "cf/api"
	"cf/configuration"
	"cf/net"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testhelpers"
	"testing"
)

var findAllRoutesResponse = testhelpers.TestResponse{Status: http.StatusOK, Body: `
{
  "resources": [
    {
      "metadata": {
        "guid": "route-1-guid"
      },
      "entity": {
        "host": "route-1-host",
        "domain": {
          "metadata": {
            "guid": "domain-1-guid"
          },
          "entity": {
            "name": "cfapps.io"
          }
        },
        "apps": [
       	  {
       	    "metadata": {
              "guid": "app-1-guid"
            },
            "entity": {
              "name": "app-1"
       	    }
       	  }
        ]
      }
    },
    {
      "metadata": {
        "guid": "route-2-guid"
      },
      "entity": {
        "host": "route-2-host",
        "domain": {
          "metadata": {
            "guid": "domain-2-guid"
          },
          "entity": {
            "name": "example.com"
          }
        },
        "apps": [
       	  {
       	    "metadata": {
              "guid": "app-2-guid"
            },
            "entity": {
              "name": "app-2"
       	    }
       	  },
       	  {
       	    "metadata": {
              "guid": "app-3-guid"
            },
            "entity": {
              "name": "app-3"
       	    }
       	  }
        ]
      }
    }
  ]
}`}

func TestRoutesFindAll(t *testing.T) {
	endpoint, status := testhelpers.CreateCheckableEndpoint(
		"GET",
		"/v2/routes?inline-relations-depth=1",
		nil,
		findAllRoutesResponse,
	)

	ts := httptest.NewTLSServer(endpoint)
	defer ts.Close()

	repo, _ := getRepo(ts.URL)
	routes, apiResponse := repo.FindAll()

	assert.True(t, status.Called())
	assert.False(t, apiResponse.IsNotSuccessful())
	assert.Equal(t, len(routes), 2)

	route := routes[0]
	assert.Equal(t, route.Host, "route-1-host")
	assert.Equal(t, route.Guid, "route-1-guid")
	assert.Equal(t, route.Domain.Name, "cfapps.io")
	assert.Equal(t, route.Domain.Guid, "domain-1-guid")
	assert.Equal(t, route.AppNames, []string{"app-1"})

	route = routes[1]
	assert.Equal(t, route.Guid, "route-2-guid")
	assert.Equal(t, route.AppNames, []string{"app-2", "app-3"})
}

var findRouteByHostResponse = testhelpers.TestResponse{Status: http.StatusCreated, Body: `
{ "resources": [
    {
    	"metadata": {
        	"guid": "my-route-guid"
    	},
    	"entity": {
       	     "host": "my-cool-app"
    	}
    }
]}`}

func TestFindByHost(t *testing.T) {
	endpoint, status := testhelpers.CreateCheckableEndpoint(
		"GET",
		"/v2/routes?q=host%3Amy-cool-app",
		nil,
		findRouteByHostResponse,
	)

	ts := httptest.NewTLSServer(endpoint)
	defer ts.Close()

	repo, _ := getRepo(ts.URL)
	route, apiResponse := repo.FindByHost("my-cool-app")

	assert.True(t, status.Called())
	assert.False(t, apiResponse.IsNotSuccessful())
	assert.Equal(t, route, cf.Route{Host: "my-cool-app", Guid: "my-route-guid"})
}

func TestFindByHostWhenHostIsNotFound(t *testing.T) {
	endpoint, status := testhelpers.CreateCheckableEndpoint(
		"GET",
		"/v2/routes?q=host%3Amy-cool-app",
		nil,
		testhelpers.TestResponse{Status: http.StatusCreated, Body: ` { "resources": [ ]}`},
	)

	ts := httptest.NewTLSServer(http.HandlerFunc(endpoint))
	defer ts.Close()

	repo, _ := getRepo(ts.URL)
	_, apiResponse := repo.FindByHost("my-cool-app")

	assert.True(t, status.Called())
	assert.True(t, apiResponse.IsNotSuccessful())
}

func TestFindByHostAndDomain(t *testing.T) {
	endpoint, status := testhelpers.CreateCheckableEndpoint(
		"GET",
		"/v2/routes?q=host%3Amy-cool-app%3Bdomain_guid%3Amy-domain-guid",
		nil,
		findRouteByHostResponse,
	)

	ts := httptest.NewTLSServer(endpoint)
	defer ts.Close()

	repo, domainRepo := getRepo(ts.URL)
	domainRepo.FindByNameDomain = cf.Domain{Guid: "my-domain-guid"}
	route, apiResponse := repo.FindByHostAndDomain("my-cool-app", "my-domain.com")

	assert.False(t, apiResponse.IsNotSuccessful())
	assert.True(t, status.Called())
	assert.Equal(t, domainRepo.FindByNameName, "my-domain.com")
	assert.Equal(t, route, cf.Route{Host: "my-cool-app", Guid: "my-route-guid", Domain: domainRepo.FindByNameDomain})
}

func TestFindByHostAndDomainWhenRouteIsNotFound(t *testing.T) {
	endpoint, status := testhelpers.CreateCheckableEndpoint(
		"GET",
		"/v2/routes?q=host%3Amy-cool-app%3Bdomain_guid%3Amy-domain-guid",
		nil,
		testhelpers.TestResponse{Status: http.StatusOK, Body: `{ "resources": [ ] }`},
	)

	ts := httptest.NewTLSServer(endpoint)
	defer ts.Close()

	repo, domainRepo := getRepo(ts.URL)
	domainRepo.FindByNameDomain = cf.Domain{Guid: "my-domain-guid"}
	_, apiResponse := repo.FindByHostAndDomain("my-cool-app", "my-domain.com")

	assert.True(t, status.Called())
	assert.False(t, apiResponse.IsError())
	assert.True(t, apiResponse.IsNotFound())
}

func TestCreateRoute(t *testing.T) {
	endpoint, status := testhelpers.CreateCheckableEndpoint(
		"POST",
		"/v2/routes",
		testhelpers.RequestBodyMatcher(`{"host":"my-cool-app","domain_guid":"my-domain-guid","space_guid":"my-space-guid"}`),
		testhelpers.TestResponse{Status: http.StatusCreated, Body: `
{
  "metadata": { "guid": "my-route-guid" },
  "entity": { "host": "my-cool-app" }
}`},
	)

	ts := httptest.NewTLSServer(endpoint)
	defer ts.Close()

	repo, _ := getRepo(ts.URL)
	domain := cf.Domain{Guid: "my-domain-guid"}
	newRoute := cf.Route{Host: "my-cool-app"}

	createdRoute, apiResponse := repo.Create(newRoute, domain)
	assert.True(t, status.Called())
	assert.False(t, apiResponse.IsNotSuccessful())

	assert.Equal(t, createdRoute, cf.Route{Host: "my-cool-app", Guid: "my-route-guid"})
}

func TestBind(t *testing.T) {
	endpoint, status := testhelpers.CreateCheckableEndpoint(
		"PUT",
		"/v2/apps/my-cool-app-guid/routes/my-cool-route-guid",
		nil,
		testhelpers.TestResponse{Status: http.StatusCreated, Body: ""},
	)

	ts := httptest.NewTLSServer(endpoint)
	defer ts.Close()

	repo, _ := getRepo(ts.URL)
	route := cf.Route{Guid: "my-cool-route-guid"}
	app := cf.Application{Guid: "my-cool-app-guid"}

	apiResponse := repo.Bind(route, app)
	assert.True(t, status.Called())
	assert.False(t, apiResponse.IsNotSuccessful())
}

func TestUnbind(t *testing.T) {
	endpoint, status := testhelpers.CreateCheckableEndpoint(
		"DELETE",
		"/v2/apps/my-cool-app-guid/routes/my-cool-route-guid",
		nil,
		testhelpers.TestResponse{Status: http.StatusCreated, Body: ""},
	)

	ts := httptest.NewTLSServer(endpoint)
	defer ts.Close()

	repo, _ := getRepo(ts.URL)
	route := cf.Route{Guid: "my-cool-route-guid"}
	app := cf.Application{Guid: "my-cool-app-guid"}

	apiResponse := repo.Unbind(route, app)
	assert.True(t, status.Called())
	assert.False(t, apiResponse.IsNotSuccessful())
}

func getRepo(targetURL string) (repo CloudControllerRouteRepository, domainRepo *testhelpers.FakeDomainRepository) {
	config := &configuration.Configuration{
		AccessToken: "BEARER my_access_token",
		Target:      targetURL,
		Space:       cf.Space{Guid: "my-space-guid"},
	}

	gateway := net.NewCloudControllerGateway()
	domainRepo = &testhelpers.FakeDomainRepository{}

	repo = NewCloudControllerRouteRepository(config, gateway, domainRepo)
	return
}
