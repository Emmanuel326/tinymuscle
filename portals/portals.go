package portals

type Portal struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    URL         string            `json:"url"`
    Goal        string            `json:"goal"`
    IntervalMin int               `json:"interval_min"`
    Headers     map[string]string `json:"headers,omitempty"`
    Cookies     map[string]string `json:"cookies,omitempty"`
}
