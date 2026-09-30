package trudvsem

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vscan/internal/httpx"
	"vscan/internal/provider"
)

func TestRequirementObject(t *testing.T) {
	const body = `{"status":"200","meta":{"total":1},"results":{"vacancies":[{"vacancy":{"job-name":"Angular","vac_url":"https://trudvsem.ru/vacancy/card/1/abc","duty":"код","requirement":{"education":"Высшее","experience":3}}}]}}`
	jobs, _, err := parse([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || !strings.Contains(jobs[0].Description, "Высшее") || !strings.Contains(jobs[0].Description, "опыт 3") {
		t.Fatalf("%#v", jobs)
	}
}

func TestSearch(t *testing.T) {
	const body = `{"status":"200","meta":{"total":1,"limit":100},"results":{"vacancies":[{"vacancy":{"job-name":"Senior Angular Developer","vac_url":"https://trudvsem.ru/vacancy/card/1/abc","creation-date":"2026-09-24","salary_min":130000,"salary_max":180000,"currency":"«руб.»","schedule":"Удалённая работа","duty":"Писать на TypeScript","requirements":"Angular","qualification":"не указано","skills":"TypeScript","company":{"name":"ООО \"Датаворкс\""},"region":{"name":"Новосибирская область"},"category":{"specialisation":"Информационные технологии"}}}]}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("text") != "Angular TypeScript" || r.URL.Query().Get("offset") != "0" || r.URL.Query().Get("limit") != "100" {
			t.Errorf("query %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()
	p := &Provider{HTTP: &httpx.Client{HTTP: srv.Client()}, URL: srv.URL}
	var got []provider.Vacancy
	if err := p.Search(context.Background(), []string{"Angular TypeScript"}, 1, func(l provider.Listing) {
		got = append(got, l.Vacancy)
	}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("%#v", got)
	}
	v := got[0]
	if v.Source != "trudvsem" || v.Remote != "yes" || v.Company != `ООО "Датаворкс"` || v.Location != "Новосибирская область" {
		t.Fatalf("%#v", v)
	}
	if v.Salary != "130000-180000 «руб.»" || v.Posted.IsZero() || len(v.Skills) != 1 {
		t.Fatalf("%#v", v)
	}
	for _, tag := range v.Tags {
		if tag == "не указано" {
			t.Fatal("blank qualification must be dropped")
		}
	}
}
