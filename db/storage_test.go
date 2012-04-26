package db

import (
	. "launchpad.net/gocheck"
	"testing"
)

func Test(t *testing.T) { TestingT(t) }

type S struct{}

var _ = Suite(&S{})

func (s *S) TearDownSuite(c *C) {
	storage, _ := Open("127.0.0.1:27017", "tsuru_storage_test")
	storage.session.DB("tsuru").DropDatabase()
	storage.Close()
}

func (s *S) TestShouldProvideMethodToOpenAConnection(c *C) {
	storage, _ := Open("127.0.0.1:27017", "tsuru_storage_test")
	c.Assert(storage.session.Ping(), IsNil)
	storage.Close()
}

func (s *S) TestMethodCloseSholdCloseTheConnectionWithMongoDB(c *C) {
	defer func() {
		if r := recover(); r == nil {
			c.Errorf("Should close the connection, but did not!")
		}
	}()
	storage, _ := Open("127.0.0.1:27017", "tsuru_storage_test")
	storage.Close()
	storage.session.Ping()
}

func (s *S) TestShouldProvidePrivateMethodToGetACollection(c *C) {
	storage, _ := Open("127.0.0.1:27017", "tsuru_storage_test")
	defer storage.Close()
	collection := storage.getCollection("users")
	c.Assert(collection.FullName, Equals, storage.dbname+".users")
}

func (s *S) TestShouldCacheCollection(c *C) {
	storage, _ := Open("127.0.0.1:27017", "tsuru_storage_test")
	defer storage.Close()
	collection := storage.getCollection("users")
	c.Assert(collection, DeepEquals, storage.collections["users"])
}

func (s *S) TestMethodUsersShouldReturnUsersCollection(c *C) {
	storage, _ := Open("127.0.0.1:27017", "tsuru_storage_test")
	defer storage.Close()
	users := storage.Users()
	usersc := storage.getCollection("users")
	c.Assert(users, DeepEquals, usersc)
}

func (s *S) TestMethodUserShouldReturnUsersCollectionWithUniqueIndexForEmail(c *C) {
	storage, _ := Open("127.0.0.1:27017", "tsuru_storage_test")
	defer storage.Close()
	users := storage.Users()
	indexes, err := users.Indexes()
	c.Assert(err, IsNil)
	found := false
	for _, index := range indexes {
		for _, key := range index.Key {
			if key == "email" {
				c.Assert(index.Unique, Equals, true)
				found = true
				break
			}
		}

		if found {
			break
		}
	}

	if !found {
		c.Errorf("Users should declare a unique index for email")
	}
}

func (s *S) TestMethodAppsShouldReturnAppsCollection(c *C) {
	storage, _ := Open("127.0.0.1:27017", "tsuru_storage_test")
	defer storage.Close()
	apps := storage.Apps()
	appsc := storage.getCollection("apps")
	c.Assert(apps, DeepEquals, appsc)
}

func (s *S) TestMethodAppsShouldReturnAppsCollectionWithUniqueIndexForName(c *C) {
	storage, _ := Open("127.0.0.1:27017", "tsuru_storage_test")
	defer storage.Close()
	apps := storage.Apps()
	indexes, err := apps.Indexes()
	c.Assert(err, IsNil)
	found := false
	for _, index := range indexes {
		for _, key := range index.Key {
			if key == "name" {
				c.Assert(index.Unique, Equals, true)
				found = true
				break
			}
		}

		if found {
			break
		}
	}

	if !found {
		c.Errorf("Apps should declare a unique index for name")
	}
}

func (s *S) TestMethodServicesShouldReturnServicesCollection(c *C) {
	storage, _ := Open("127.0.0.1:27017", "tsuru_storage_test")
	defer storage.Close()
	services := storage.Services()
	servicesc := storage.getCollection("services")
	c.Assert(services, DeepEquals, servicesc)
}

func (s *S) TestMethodServiceAppsShouldReturnServiceAppsCollection(c *C) {
	storage, _ := Open("127.0.0.1:27017", "tsuru_storage_test")
	defer storage.Close()
	serviceApps := storage.ServiceApps()
	serviceAppsc := storage.getCollection("service_apps")
	c.Assert(serviceApps, DeepEquals, serviceAppsc)
}

func (s *S) TestMethodServiceTypesReturnServiceTypesCollection(c *C) {
	storage, _ := Open("127.0.0.1:27017", "tsuru_storage_test")
	defer storage.Close()
	serviceTypes := storage.ServiceTypes()
	serviceTypesc := storage.getCollection("service_types")
	c.Assert(serviceTypes, DeepEquals, serviceTypesc)
}

func (s *S) TestMethodUnitsShouldReturnUnitsCollection(c *C) {
	storage, _ := Open("127.0.0.1:27017", "tsuru_storage_test")
	defer storage.Close()
	units := storage.Units()
	unitsc := storage.getCollection("units")
	c.Assert(units, DeepEquals, unitsc)
}