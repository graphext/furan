package tagcheck

type FakeChecker struct {
	AllTagsExistFunc func(tags []string, repo string) (bool, []string, error)
}

func (fc *FakeChecker) AllTagsExist(tags []string, repo string) (bool, []string, error) {
	if fc.AllTagsExistFunc != nil {
		return fc.AllTagsExistFunc(tags, repo)
	}
	return false, nil, nil
}
