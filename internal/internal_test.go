package internal

/*
func TestPasswordType(t *testing.T) {
	t.Parallel()

	data := `"thispasswordiscomplex"`

	var pw dto.Password
	err := json.NewDecoder(strings.NewReader(data)).Decode(&pw)
	if err != nil {
		t.Fatal(err)
	}

	if exp := data[1 : len(data)-1]; pw != dto.Password(exp) {
		t.Errorf(`expected %s got %s`, exp, pw)
	}

	hash, err := pw.Hash()
	if err != nil {
		t.Fatal(err)
	}

	err = hash.Compare(pw)
	if err != nil {
		t.Fatal(err)
	}

	err = hash.Compare("1234")
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestEmailType(t *testing.T) {
	t.Parallel()

	data := `"anon@gmail.com"`

	var e dto.Email
	err := json.NewDecoder(strings.NewReader(data)).Decode(&e)
	if err != nil {
		t.Fatal(err)
	}

	var sb strings.Builder
	err = json.NewEncoder(&sb).Encode(e)
	if err != nil {
		t.Fatal(err)
	}

	if s := strings.TrimSpace(sb.String()); s[1:len(s)-1] != string(e) {
		t.Fatalf("expected %s got %s", e, s)
	}

	// should return an error
	data = `"anon@.gmail.com"`

	err = json.NewDecoder(strings.NewReader(data)).Decode(&e)
	if err == nil {
		t.Errorf("expected %v", dto.ErrInvalidEmail)
	}
}
*/
