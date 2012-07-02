package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/timeredbull/tsuru/api/app"
	"github.com/timeredbull/tsuru/api/auth"
	"github.com/timeredbull/tsuru/db"
	"github.com/timeredbull/tsuru/errors"
	"io/ioutil"
	"labix.org/v2/mgo/bson"
	. "launchpad.net/gocheck"
	"net/http"
	"net/http/httptest"
	"path/filepath"
)

func makeRequestToCreateHandler(c *C) (*httptest.ResponseRecorder, *http.Request) {
	manifest := `id: some_service
endpoint:
    production: someservice.com
    test: test.someservice.com
`
	b := bytes.NewBufferString(manifest)
	request, err := http.NewRequest("POST", "/services", b)
	c.Assert(err, IsNil)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	return recorder, request
}

func makeRequestToCreateInstanceHandler(c *C) (*httptest.ResponseRecorder, *http.Request) {
	b := bytes.NewBufferString(`{"name": "brainSQL", "service_name": "mysql"}`)
	request, err := http.NewRequest("POST", "/services/instances", b)
	c.Assert(err, IsNil)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	return recorder, request
}

func (s *S) TestCreateHandlerSavesNameFromManifestId(c *C) {
	recorder, request := makeRequestToCreateHandler(c)
	err := CreateHandler(recorder, request, s.user)
	c.Assert(err, IsNil)
	query := bson.M{"_id": "some_service"}
	var rService Service
	err = db.Session.Services().Find(query).One(&rService)
	c.Assert(err, IsNil)
	c.Assert(rService.Name, Equals, "some_service")
}

func (s *S) TestCreateHandlerSavesEndpointServiceProperty(c *C) {
	recorder, request := makeRequestToCreateHandler(c)
	err := CreateHandler(recorder, request, s.user)
	c.Assert(err, IsNil)
	query := bson.M{"_id": "some_service"}
	var rService Service
	err = db.Session.Services().Find(query).One(&rService)
	c.Assert(err, IsNil)
	c.Assert(rService.Endpoint["production"], Equals, "someservice.com")
	c.Assert(rService.Endpoint["test"], Equals, "test.someservice.com")
}

func (s *S) TestCreateHandlerWithContentOfRealYaml(c *C) {
	p, err := filepath.Abs("testdata/manifest.yml")
	manifest, err := ioutil.ReadFile(p)
	c.Assert(err, IsNil)
	b := bytes.NewBuffer(manifest)
	request, err := http.NewRequest("POST", "/services", b)
	c.Assert(err, IsNil)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	err = CreateHandler(recorder, request, s.user)
	c.Assert(err, IsNil)
	query := bson.M{"_id": "mysqlapi"}
	var rService Service
	err = db.Session.Services().Find(query).One(&rService)
	c.Assert(err, IsNil)
	c.Assert(rService.Endpoint["production"], Equals, "mysqlapi.com")
	c.Assert(rService.Endpoint["test"], Equals, "localhost:8000")
}

func (s *S) TestCreateHandlerShouldReturnErrorWhenNameExists(c *C) {
	recorder, request := makeRequestToCreateHandler(c)
	err := CreateHandler(recorder, request, s.user)
	c.Assert(err, IsNil)
	recorder, request = makeRequestToCreateHandler(c)
	err = CreateHandler(recorder, request, s.user)
	c.Assert(err, Not(IsNil))
	c.Assert(err, ErrorMatches, "^Service with name some_service already exists.$")
}

func (s *S) TestCreateHandlerGetAllTeamsFromTheUser(c *C) {
	recorder, request := makeRequestToCreateHandler(c)
	err := CreateHandler(recorder, request, s.user)
	c.Assert(err, IsNil)
	c.Assert(recorder.Body.String(), Equals, "success")
	c.Assert(recorder.Code, Equals, 200)
	query := bson.M{"_id": "some_service"}
	var rService Service
	err = db.Session.Services().Find(query).One(&rService)
	c.Assert(err, IsNil)
	c.Assert(rService.Name, Equals, "some_service")
	c.Assert(*s.team, HasAccessTo, rService)
}

func (s *S) TestCreateHandlerReturnsForbiddenIfTheUserIsNotMemberOfAnyTeam(c *C) {
	u := &auth.User{Email: "enforce@queensryche.com", Password: "123"}
	u.Create()
	defer db.Session.Users().RemoveAll(bson.M{"email": u.Email})
	recorder, request := makeRequestToCreateHandler(c)
	err := CreateHandler(recorder, request, u)
	c.Assert(err, NotNil)
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusForbidden)
	c.Assert(e, ErrorMatches, "^In order to create a service, you should be member of at least one team$")
}

func (suite *S) TestCreateInstanceHandlerSavesServiceInstanceInDb(c *C) {
	s := Service{Name: "mysql", Teams: []string{suite.team.Name}}
	s.Create()
	recorder, request := makeRequestToCreateInstanceHandler(c)
	err := CreateInstanceHandler(recorder, request, suite.user)
	c.Assert(err, IsNil)
	var si ServiceInstance
	db.Session.ServiceInstances().Find(bson.M{"_id": "brainSQL", "service_name": "mysql"}).One(&si)
	c.Assert(si.Name, Equals, "brainSQL")
	c.Assert(si.ServiceName, Equals, "mysql")
}

func (s *S) TestCreateInstanceHandlerReturnsErrorWhenUserCannotUseService(c *C) {
	service := Service{Name: "mysql"}
	service.Create()
	recorder, request := makeRequestToCreateInstanceHandler(c)
	err := CreateInstanceHandler(recorder, request, s.user)
	c.Assert(err, ErrorMatches, "^You don't have access to service mysql$")
}

func (s *S) TestCreateInstanceHandlerReturnsErrorWhenServiceDoesntExists(c *C) {
	recorder, request := makeRequestToCreateInstanceHandler(c)
	err := CreateInstanceHandler(recorder, request, s.user)
	c.Assert(err, ErrorMatches, "^Service mysql does not exists.$")
}

func (s *S) TestCreateInstanceHandlerCreatesVMInstanceWhenServicesManifestIsConfiguredToDoSo(c *C) {
	service := Service{Name: "mysql", Teams: []string{s.team.Name}}
	service.Create()
	recorder, request := makeRequestToCreateInstanceHandler(c)
	err := CreateInstanceHandler(recorder, request, s.user)
	c.Assert(err, IsNil)
	// #TODO finish me!!
}

func (s *S) TestDeleteHandler(c *C) {
	se := Service{Name: "Mysql", Teams: []string{s.team.Name}}
	se.Create()
	defer se.Delete()
	request, err := http.NewRequest("DELETE", fmt.Sprintf("/services/%s?:name=%s", se.Name, se.Name), nil)
	c.Assert(err, IsNil)
	recorder := httptest.NewRecorder()
	err = DeleteHandler(recorder, request, s.user)
	c.Assert(err, IsNil)
	c.Assert(recorder.Code, Equals, 200)
	query := bson.M{"_id": "Mysql"}
	qtd, err := db.Session.Services().Find(query).Count()
	c.Assert(err, IsNil)
	c.Assert(qtd, Equals, 0)
}

func (s *S) TestDeleteHandlerReturns404(c *C) {
	request, err := http.NewRequest("DELETE", fmt.Sprintf("/services/%s?:name=%s", "mongodb", "mongodb"), nil)
	c.Assert(err, IsNil)
	recorder := httptest.NewRecorder()
	err = DeleteHandler(recorder, request, s.user)
	c.Assert(err, NotNil)
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusNotFound)
	c.Assert(e, ErrorMatches, "^Service not found$")
}

func (s *S) TestDeleteHandlerReturns403IfTheUserDoesNotHaveAccessToTheService(c *C) {
	se := Service{Name: "Mysql"}
	se.Create()
	defer se.Delete()
	request, err := http.NewRequest("DELETE", fmt.Sprintf("/services/%s?:name=%s", se.Name, se.Name), nil)
	c.Assert(err, IsNil)
	recorder := httptest.NewRecorder()
	err = DeleteHandler(recorder, request, s.user)
	c.Assert(err, NotNil)
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusForbidden)
	c.Assert(e, ErrorMatches, "^This user does not have access to this service$")
}

// func (s *S) TestBindHandler(c *C) {
// 	st := ServiceType{Name: "Mysql", Charm: "mysql"}
// 	err := st.Create()
// 	c.Assert(err, IsNil)
// 	se := Service{ServiceTypeName: st.Name, Name: "my_service", Teams: []string{s.team.Name}}
// 	a := app.App{Name: "serviceApp", Framework: "django", Teams: []string{s.team.Name}}
// 	err = se.Create()
// 	c.Assert(err, IsNil)
// 	err = a.Create()
// 	c.Assert(err, IsNil)
// 	b := strings.NewReader(`{"app":"serviceApp", "service":"my_service"}`)
// 	request, err := http.NewRequest("POST", "/services/bind", b)
// 	c.Assert(err, IsNil)
// 	recorder := httptest.NewRecorder()
// 	err = BindHandler(recorder, request, s.user)
// 	c.Assert(err, IsNil)
// 	c.Assert(recorder.Code, Equals, 200)
// 	query := bson.M{
// 		"service_name": se.Name,
// 		"app_name":     a.Name,
// 	}
// 	qtd, err := db.Session.ServiceInstances().Find(query).Count()
// 	c.Check(err, IsNil)
// 	c.Assert(qtd, Equals, 1)
// }
// 
// func (s *S) TestBindHandlerReturns403IfTheUserDoesNotHaveAccessToTheApp(c *C) {
// 	st := ServiceType{Name: "Mysql", Charm: "mysql"}
// 	err := st.Create()
// 	c.Assert(err, IsNil)
// 	se := Service{ServiceTypeName: st.Name, Name: "my_service", Teams: []string{s.team.Name}}
// 	a := app.App{Name: "serviceApp", Framework: "django"}
// 	err = se.Create()
// 	c.Assert(err, IsNil)
// 	err = a.Create()
// 	c.Assert(err, IsNil)
// 	b := strings.NewReader(`{"app":"serviceApp", "service":"my_service"}`)
// 	request, err := http.NewRequest("POST", "/services/bind", b)
// 	c.Assert(err, IsNil)
// 	recorder := httptest.NewRecorder()
// 	err = BindHandler(recorder, request, s.user)
// 	c.Assert(err, NotNil)
// 	e, ok := err.(*errors.Http)
// 	c.Assert(ok, Equals, true)
// 	c.Assert(e.Code, Equals, http.StatusForbidden)
// 	c.Assert(e, ErrorMatches, "^This user does not have access to this app$")
// }

// func (s *S) TestBindHandlerReturns404IfTheAppDoesNotExist(c *C) {
// 	st := ServiceType{Name: "Mysql", Charm: "mysql"}
// 	err := st.Create()
// 	c.Assert(err, IsNil)
// 	se := Service{ServiceTypeName: st.Name, Name: "my_service", Teams: []string{s.team.Name}}
// 	err = se.Create()
// 	c.Assert(err, IsNil)
// 	b := strings.NewReader(`{"app":"serviceApp", "service":"my_service"}`)
// 	request, err := http.NewRequest("POST", "/services/bind", b)
// 	c.Assert(err, IsNil)
// 	recorder := httptest.NewRecorder()
// 	err = BindHandler(recorder, request, s.user)
// 	c.Assert(err, NotNil)
// 	e, ok := err.(*errors.Http)
// 	c.Assert(ok, Equals, true)
// 	c.Assert(e.Code, Equals, http.StatusNotFound)
// 	c.Assert(e, ErrorMatches, "^App not found$")
// }
// 
// func (s *S) TestBindHandlerReturns403IfTheUserDoesNotHaveAccessToTheService(c *C) {
// 	st := ServiceType{Name: "Mysql", Charm: "mysql"}
// 	err := st.Create()
// 	c.Assert(err, IsNil)
// 	se := Service{ServiceTypeName: st.Name, Name: "my_service"}
// 	a := app.App{Name: "serviceApp", Framework: "django", Teams: []string{s.team.Name}}
// 	err = se.Create()
// 	c.Assert(err, IsNil)
// 	err = a.Create()
// 	c.Assert(err, IsNil)
// 	b := strings.NewReader(`{"app":"serviceApp", "service":"my_service"}`)
// 	request, err := http.NewRequest("POST", "/services/bind", b)
// 	c.Assert(err, IsNil)
// 	recorder := httptest.NewRecorder()
// 	err = BindHandler(recorder, request, s.user)
// 	c.Assert(err, NotNil)
// 	e, ok := err.(*errors.Http)
// 	c.Assert(ok, Equals, true)
// 	c.Assert(e.Code, Equals, http.StatusForbidden)
// 	c.Assert(e, ErrorMatches, "^This user does not have access to this service$")
// }

// func (s *S) TestBindHandlerReturns404IfTheServiceDoesNotExist(c *C) {
// 	a := app.App{Name: "serviceApp", Framework: "django", Teams: []string{s.team.Name}}
// 	err := a.Create()
// 	c.Assert(err, IsNil)
// 	b := strings.NewReader(`{"app":"serviceApp", "service":"my_service"}`)
// 	request, err := http.NewRequest("POST", "/services/bind", b)
// 	c.Assert(err, IsNil)
// 	recorder := httptest.NewRecorder()
// 	err = BindHandler(recorder, request, s.user)
// 	c.Assert(err, NotNil)
// 	e, ok := err.(*errors.Http)
// 	c.Assert(ok, Equals, true)
// 	c.Assert(e.Code, Equals, http.StatusNotFound)
// 	c.Assert(e, ErrorMatches, "^Service not found$")
// 
// }

// func (s *S) TestUnbindHandler(c *C) {
// 	st := ServiceType{Name: "Mysql", Charm: "mysql"}
// 	st.Create()
// 	se := Service{ServiceTypeName: st.Name, Name: "my_service", Teams: []string{s.team.Name}}
// 	a := app.App{Name: "serviceApp", Framework: "django", Teams: []string{s.team.Name}}
// 	se.Create()
// 	a.Create()
// 	se.Bind(&a)
// 	b := strings.NewReader(`{"app":"serviceApp", "service":"my_service"}`)
// 	request, err := http.NewRequest("POST", "/services/bind", b)
// 	c.Assert(err, IsNil)
// 	recorder := httptest.NewRecorder()
// 	err = UnbindHandler(recorder, request, s.user)
// 	c.Assert(err, IsNil)
// 	c.Assert(recorder.Code, Equals, 200)
// 	query := bson.M{
// 		"service_name": se.Name,
// 		"app_name":     a.Name,
// 	}
// 	qtd, err := db.Session.Services().Find(query).Count()
// 	c.Check(err, IsNil)
// 	c.Assert(qtd, Equals, 0)
// }

// func (s *S) TestUnbindHandlerReturns403IfTheUserDoesNotHaveAccessToTheService(c *C) {
// 	st := ServiceType{Name: "Mysql", Charm: "mysql"}
// 	st.Create()
// 	se := Service{ServiceTypeName: st.Name, Name: "my_service"}
// 	a := app.App{Name: "serviceApp", Framework: "django", Teams: []string{s.team.Name}}
// 	se.Create()
// 	a.Create()
// 	se.Bind(&a)
// 	b := strings.NewReader(`{"app":"serviceApp", "service":"my_service"}`)
// 	request, err := http.NewRequest("POST", "/services/bind", b)
// 	c.Assert(err, IsNil)
// 	recorder := httptest.NewRecorder()
// 	err = UnbindHandler(recorder, request, s.user)
// 	c.Assert(err, NotNil)
// 	e, ok := err.(*errors.Http)
// 	c.Assert(ok, Equals, true)
// 	c.Assert(e.Code, Equals, http.StatusForbidden)
// 	c.Assert(e, ErrorMatches, "^This user does not have access to this service$")
// }

// func (s *S) TestUnbindHandlerReturns404IfTheServiceDoesNotExist(c *C) {
// 	st := ServiceType{Name: "Mysql", Charm: "mysql"}
// 	st.Create()
// 	a := app.App{Name: "serviceApp", Framework: "django", Teams: []string{*s.team.Name}}
// 	a.Create()
// 	b := strings.NewReader(`{"app":"serviceApp", "service":"my_service"}`)
// 	request, err := http.NewRequest("POST", "/services/bind", b)
// 	c.Assert(err, IsNil)
// 	recorder := httptest.NewRecorder()
// 	err = UnbindHandler(recorder, request, s.user)
// 	c.Assert(err, NotNil)
// 	e, ok := err.(*errors.Http)
// 	c.Assert(ok, Equals, true)
// 	c.Assert(e.Code, Equals, http.StatusNotFound)
// 	c.Assert(e, ErrorMatches, "^Service not found$")
// }

// func (s *S) TestUnbindHandlerReturns403IfTheUserDoesNotHaveAccessToTheApp(c *C) {
// 	st := ServiceType{Name: "Mysql", Charm: "mysql"}
// 	st.Create()
// 	se := Service{ServiceTypeName: st.Name, Name: "my_service", Teams: []string{s.team.Name}}
// 	a := app.App{Name: "serviceApp", Framework: "django"}
// 	se.Create()
// 	a.Create()
// 	se.Bind(&a)
// 	b := strings.NewReader(`{"app":"serviceApp", "service":"my_service"}`)
// 	request, err := http.NewRequest("POST", "/services/bind", b)
// 	c.Assert(err, IsNil)
// 	recorder := httptest.NewRecorder()
// 	err = UnbindHandler(recorder, request, s.user)
// 	c.Assert(err, NotNil)
// 	e, ok := err.(*errors.Http)
// 	c.Assert(ok, Equals, true)
// 	c.Assert(e.Code, Equals, http.StatusForbidden)
// 	c.Assert(e, ErrorMatches, "^This user does not have access to this app$")
// }

// func (s *S) TestUnbindHandlerReturns404IfTheAppDoesNotExist(c *C) {
// 	st := ServiceType{Name: "Mysql", Charm: "mysql"}
// 	st.Create()
// 	se := Service{ServiceTypeName: st.Name, Name: "my_service", Teams: []string{s.team.Name}}
// 	se.Create()
// 	b := strings.NewReader(`{"app":"serviceApp", "service":"my_service"}`)
// 	request, err := http.NewRequest("POST", "/services/bind", b)
// 	c.Assert(err, IsNil)
// 	recorder := httptest.NewRecorder()
// 	err = UnbindHandler(recorder, request, s.user)
// 	c.Assert(err, NotNil)
// 	e, ok := err.(*errors.Http)
// 	c.Assert(ok, Equals, true)
// 	c.Assert(e.Code, Equals, http.StatusNotFound)
// 	c.Assert(e, ErrorMatches, "^App not found$")
// }

func (s *S) TestGrantAccessToTeam(c *C) {
	t := &auth.Team{Name: "blaaaa"}
	db.Session.Teams().Insert(t)
	defer db.Session.Teams().Remove(bson.M{"name": t.Name})
	se := Service{Name: "my_service", Teams: []string{s.team.Name}}
	err := se.Create()
	defer se.Delete()
	c.Assert(err, IsNil)
	url := fmt.Sprintf("/services/%s/%s?:service=%s&:team=%s", se.Name, t.Name, se.Name, t.Name)
	request, err := http.NewRequest("PUT", url, nil)
	c.Assert(err, IsNil)
	recorder := httptest.NewRecorder()
	err = GrantAccessToTeamHandler(recorder, request, s.user)
	c.Assert(err, IsNil)
	err = se.Get()
	c.Assert(err, IsNil)
	c.Assert(*s.team, HasAccessTo, se)
}

func (s *S) TestGrantAccesToTeamReturnNotFoundIfTheServiceDoesNotExist(c *C) {
	url := fmt.Sprintf("/services/nononono/%s?:service=nononono&:team=%s", s.team.Name, s.team.Name)
	request, err := http.NewRequest("PUT", url, nil)
	c.Assert(err, IsNil)
	recorder := httptest.NewRecorder()
	err = GrantAccessToTeamHandler(recorder, request, s.user)
	c.Assert(err, NotNil)
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusNotFound)
	c.Assert(e, ErrorMatches, "^Service not found$")
}

func (s *S) TestGrantAccessToTeamReturnForbiddenIfTheGivenUserDoesNotHaveAccessToTheService(c *C) {
	se := Service{Name: "my_service"}
	err := se.Create()
	c.Assert(err, IsNil)
	defer se.Delete()
	url := fmt.Sprintf("/services/%s/%s?:service=%s&:team=%s", se.Name, s.team.Name, se.Name, s.team.Name)
	request, err := http.NewRequest("PUT", url, nil)
	c.Assert(err, IsNil)
	recorder := httptest.NewRecorder()
	err = GrantAccessToTeamHandler(recorder, request, s.user)
	c.Assert(err, NotNil)
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusForbidden)
	c.Assert(e, ErrorMatches, "^This user does not have access to this service$")
}

func (s *S) TestGrantAccessToTeamReturnNotFoundIfTheTeamDoesNotExist(c *C) {
	se := Service{Name: "my_service", Teams: []string{s.team.Name}}
	err := se.Create()
	c.Assert(err, IsNil)
	defer se.Delete()
	url := fmt.Sprintf("/services/%s/nonono?:service=%s&:team=nonono", se.Name, se.Name)
	request, err := http.NewRequest("PUT", url, nil)
	c.Assert(err, IsNil)
	recorder := httptest.NewRecorder()
	err = GrantAccessToTeamHandler(recorder, request, s.user)
	c.Assert(err, NotNil)
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusNotFound)
	c.Assert(e, ErrorMatches, "^Team not found$")
}

func (s *S) TestGrantAccessToTeamReturnConflictIfTheTeamAlreadyHasAccessToTheService(c *C) {
	se := Service{Name: "my_service", Teams: []string{s.team.Name}}
	err := se.Create()
	defer se.Delete()
	c.Assert(err, IsNil)
	url := fmt.Sprintf("/services/%s/%s?:service=%s&:team=%s", se.Name, s.team.Name, se.Name, s.team.Name)
	request, err := http.NewRequest("PUT", url, nil)
	c.Assert(err, IsNil)
	recorder := httptest.NewRecorder()
	err = GrantAccessToTeamHandler(recorder, request, s.user)
	c.Assert(err, NotNil)
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusConflict)
}

func (s *S) TestRevokeAccessFromTeamRemovesTeamFromService(c *C) {
	t := &auth.Team{Name: "alle-da"}
	se := Service{Name: "my_service", Teams: []string{s.team.Name, t.Name}}
	err := se.Create()
	c.Assert(err, IsNil)
	defer se.Delete()
	url := fmt.Sprintf("/services/%s/%s?:service=%s&:team=%s", se.Name, s.team.Name, se.Name, s.team.Name)
	request, err := http.NewRequest("DELETE", url, nil)
	c.Assert(err, IsNil)
	recorder := httptest.NewRecorder()
	err = RevokeAccessFromTeamHandler(recorder, request, s.user)
	c.Assert(err, IsNil)
	err = se.Get()
	c.Assert(err, IsNil)
	c.Assert(*s.team, Not(HasAccessTo), se)
}

func (s *S) TestRevokeAccessFromTeamReturnsNotFoundIfTheServiceDoesNotExist(c *C) {
	url := fmt.Sprintf("/services/nonono/%s?:service=nonono&:team=%s", s.team.Name, s.team.Name)
	request, err := http.NewRequest("DELETE", url, nil)
	c.Assert(err, IsNil)
	recorder := httptest.NewRecorder()
	err = RevokeAccessFromTeamHandler(recorder, request, s.user)
	c.Assert(err, NotNil)
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusNotFound)
	c.Assert(e, ErrorMatches, "^Service not found$")
}

func (s *S) TestRevokeAccesFromTeamReturnsForbiddenIfTheGivenUserDoesNotHasAccessToTheService(c *C) {
	t := &auth.Team{Name: "alle-da"}
	se := Service{Name: "my_service", Teams: []string{t.Name}}
	err := se.Create()
	c.Assert(err, IsNil)
	defer se.Delete()
	url := fmt.Sprintf("/services/%s/%s?:service=%s&:team=%s", se.Name, t.Name, se.Name, t.Name)
	request, err := http.NewRequest("DELETE", url, nil)
	c.Assert(err, IsNil)
	recorder := httptest.NewRecorder()
	err = RevokeAccessFromTeamHandler(recorder, request, s.user)
	c.Assert(err, NotNil)
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusForbidden)
	c.Assert(e, ErrorMatches, "^This user does not have access to this service$")
}

func (s *S) TestRevokeAccessFromTeamReturnsNotFoundIfTheTeamDoesNotExist(c *C) {
	se := Service{Name: "my_service", Teams: []string{s.team.Name}}
	err := se.Create()
	c.Assert(err, IsNil)
	defer se.Delete()
	url := fmt.Sprintf("/services/%s/nonono?:service=%s&:team=nonono", se.Name, se.Name)
	request, err := http.NewRequest("DELETE", url, nil)
	c.Assert(err, IsNil)
	recorder := httptest.NewRecorder()
	err = RevokeAccessFromTeamHandler(recorder, request, s.user)
	c.Assert(err, NotNil)
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusNotFound)
	c.Assert(e, ErrorMatches, "^Team not found$")
}

func (s *S) TestRevokeAccessFromTeamReturnsForbiddenIfTheTeamIsTheOnlyWithAccessToTheService(c *C) {
	se := Service{Name: "my_service", Teams: []string{s.team.Name}}
	err := se.Create()
	c.Assert(err, IsNil)
	defer se.Delete()
	url := fmt.Sprintf("/services/%s/%s?:service=%s&:team=%s", se.Name, s.team.Name, se.Name, s.team.Name)
	request, err := http.NewRequest("DELETE", url, nil)
	c.Assert(err, IsNil)
	recorder := httptest.NewRecorder()
	err = RevokeAccessFromTeamHandler(recorder, request, s.user)
	c.Assert(err, NotNil)
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusForbidden)
	c.Assert(e, ErrorMatches, "^You can not revoke the access from this team, because it is the unique team with access to this service, and a service can not be orphaned$")
}

func (s *S) TestRevokeAccessFromTeamReturnNotFoundIfTheTeamDoesNotHasAccessToTheService(c *C) {
	t := &auth.Team{Name: "Rammlied"}
	db.Session.Teams().Insert(t)
	defer db.Session.Teams().RemoveAll(bson.M{"name": t.Name})
	se := Service{Name: "my_service", Teams: []string{s.team.Name, s.team.Name}}
	err := se.Create()
	c.Assert(err, IsNil)
	defer se.Delete()
	url := fmt.Sprintf("/services/%s/%s?:service=%s&:team=%s", se.Name, t.Name, se.Name, t.Name)
	request, err := http.NewRequest("DELETE", url, nil)
	c.Assert(err, IsNil)
	recorder := httptest.NewRecorder()
	err = RevokeAccessFromTeamHandler(recorder, request, s.user)
	c.Assert(err, NotNil)
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusNotFound)
}

func (s *S) TestServicesHandler(c *C) {
	app := app.App{Name: "globo", Teams: []string{s.team.Name}}
	err := app.Create()
	c.Assert(err, IsNil)
	service := Service{Name: "redis", Teams: []string{s.team.Name}}
	err = service.Create()
	c.Assert(err, IsNil)
	instance := ServiceInstance{
		Name:        "redis-globo",
		ServiceName: "redis",
		Apps:        []string{"globo"},
	}
	err = instance.Create()
	c.Assert(err, IsNil)
	request, err := http.NewRequest("GET", "/services/instances", nil)
	c.Assert(err, IsNil)
	recorder := httptest.NewRecorder()
	err = ServicesHandler(recorder, request, s.user)
	c.Assert(err, IsNil)
	body, err := ioutil.ReadAll(recorder.Body)
	c.Assert(err, IsNil)
	var instances map[string][]string
	err = json.Unmarshal(body, &instances)
	c.Assert(err, IsNil)
	expected := map[string][]string{
		"redis": []string{"redis-globo"},
	}
	c.Assert(instances, DeepEquals, expected)
}

func (s *S) TestServicesHandlerReturnsOnlyServicesThatTheUserHasAccess(c *C) {
	u := &auth.User{Email: "me@globo.com", Password: "123"}
	err := u.Create()
	c.Assert(err, IsNil)
	defer db.Session.Users().Remove(bson.M{"email": u.Email})
	app := app.App{Name: "globo", Teams: []string{s.team.Name}}
	err = app.Create()
	c.Assert(err, IsNil)
	service := Service{Name: "redis"}
	err = service.Create()
	c.Assert(err, IsNil)
	instance := ServiceInstance{
		Name:        "redis-globo",
		ServiceName: "redis",
		Apps:        []string{"globo"},
	}
	err = instance.Create()
	c.Assert(err, IsNil)
	request, err := http.NewRequest("GET", "/services/instances", nil)
	c.Assert(err, IsNil)
	recorder := httptest.NewRecorder()
	err = ServicesHandler(recorder, request, u)
	c.Assert(err, IsNil)
	body, err := ioutil.ReadAll(recorder.Body)
	c.Assert(err, IsNil)
	var instances map[string][]string
	err = json.Unmarshal(body, &instances)
	c.Assert(err, IsNil)
	c.Assert(instances, DeepEquals, map[string][]string(nil))
}

func (s *S) TestServicesHandlerFilterInstancesPerServiceIncludingServicesThatDoesNotHaveInstances(c *C) {
	u := &auth.User{Email: "me@globo.com", Password: "123"}
	err := u.Create()
	c.Assert(err, IsNil)
	defer db.Session.Users().Remove(bson.M{"email": u.Email})
	app := app.App{Name: "globo", Teams: []string{s.team.Name}}
	err = app.Create()
	c.Assert(err, IsNil)
	serviceNames := []string{"redis", "mysql", "pgsql", "memcached"}
	defer db.Session.Services().RemoveAll(bson.M{"name": bson.M{"$in": serviceNames}})
	defer db.Session.ServiceInstances().RemoveAll(bson.M{"service_name": bson.M{"$in": serviceNames}})
	for _, name := range serviceNames {
		service := Service{Name: name, Teams: []string{s.team.Name}}
		err = service.Create()
		c.Assert(err, IsNil)
		instance := ServiceInstance{
			Name:        service.Name + app.Name + "1",
			ServiceName: service.Name,
			Apps:        []string{app.Name},
		}
		err = instance.Create()
		c.Assert(err, IsNil)
		instance = ServiceInstance{
			Name:        service.Name + app.Name + "2",
			ServiceName: service.Name,
			Apps:        []string{app.Name},
		}
		err = instance.Create()
	}
	service := Service{Name: "oracle", Teams: []string{s.team.Name}}
	err = service.Create()
	c.Assert(err, IsNil)
	defer db.Session.Services().Remove(bson.M{"name": "oracle"})
	request, err := http.NewRequest("GET", "/services/instances", nil)
	c.Assert(err, IsNil)
	recorder := httptest.NewRecorder()
	err = ServicesHandler(recorder, request, s.user)
	c.Assert(err, IsNil)
	body, err := ioutil.ReadAll(recorder.Body)
	c.Assert(err, IsNil)
	var instances map[string][]string
	err = json.Unmarshal(body, &instances)
	c.Assert(err, IsNil)
	expected := map[string][]string{
		"redis":     []string{"redisglobo1", "redisglobo2"},
		"mysql":     []string{"mysqlglobo1", "mysqlglobo2"},
		"pgsql":     []string{"pgsqlglobo1", "pgsqlglobo2"},
		"memcached": []string{"memcachedglobo1", "memcachedglobo2"},
		"oracle":    []string{},
	}
	c.Assert(instances, DeepEquals, expected)
}
