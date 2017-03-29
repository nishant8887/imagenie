$(document).ready(function() {
    var MainView = {
        loggedIn: ko.observable(false),
        signInBox: ko.observable(true),
        uploadBox: ko.observable(false),
        uploading: ko.observable(false),

        cuUsername: ko.observable(''),
        cuFirstname: ko.observable(''),
        cuLastname: ko.observable(''),
        cuEmail: ko.observable(''),
        cuPassword: ko.observable(''),
        cuConfirmPassword: ko.observable(''),
        cuError: ko.observable(''),

        luUsername: ko.observable(''),
        luPassword: ko.observable(''),
        luError: ko.observable(''),

        uffile: ko.observable(''),
        ufdescription: ko.observable(''),

        images: ko.observableArray([]),
        page: ko.observable(1),
        total_pages: ko.observable(1),

        showSignIn: function () {
            this.signInBox(true);
            this.luError('');
        },

        showSignUp: function() {
            this.signInBox(false);
            this.cuError('');
        },

        doLogIn: function() {
            var username = this.luUsername();
            var password = this.luPassword();
            
            if (username == "" || password == "") {
                console.log("Empty fields");
                return;
            }

            var req = {
                username: username,
                password: password
            };

            console.log(req);

            var _this = this;
            $.ajax({
                method: "POST",
                url: "/user/login",
                data: req,
                success: function(e) {
                    console.log(e);
                    _this.luPassword('');
                    _this.luError('');
                    _this.loggedIn(true);
                },
                error: function(e) {
                    _this.luError("Invalid username or password");
                }
            });
        },

        doCreateAccount: function() {
            var password = this.cuPassword();
            var confirm_password = this.cuConfirmPassword();
            var username = this.cuUsername();
            var firstname = this.cuFirstname();
            var lastname = this.cuLastname();
            var email = this.cuEmail();

            if (username == "" || firstname == "" || lastname == "" || email == "" || password == "") {
                this.cuError("All fields are required");
                return;
            }

            if (password != confirm_password) {
                this.cuError("Passwords do not match");
                return;
            }

            var req = {
                username: username,
                firstname: firstname,
                lastname: lastname,
                email: email,
                password: password
            };

            console.log(req);

            var _this = this;
            $.ajax({
                method: "POST",
                url: "/user/create",
                data: req,
                success: function(e) {
                    console.log(e);
                    _this.cuUsername('');
                    _this.cuPassword('');
                    _this.cuConfirmPassword('');
                    _this.cuFirstname('');
                    _this.cuLastname('');
                    _this.cuEmail('');
                    _this.cuError('');
                    _this.loggedIn(true);
                },
                error: function(e) {
                    _this.cuError("Username or email already exists");
                }
            });
        },

        logOut: function() {
            var _this = this;
            $.ajax({
                method: "POST",
                url: "/user/logout",
                success: function(e) {
                    console.log(e);
                    _this.loggedIn(false);
                },
                error: function(e) {
                }
            });
        },

        toggleUpload: function() {
            if (!this.loggedIn()) {
                this.uploadBox(false);
                return;
            }
            if (this.uploadBox()) {
                this.uploadBox(false);
            } else {
                this.uffile('');
                this.ufdescription('');
                this.uploadBox(true);
            }
        },

        doneUpload: function() {
            var _this = this;
            this.uploading(true);
            $("#upload_target").load(function () {
                iframeContents = this.contentWindow.document.body.innerHTML;
                if (iframeContents.indexOf("Error") !== -1 || iframeContents.indexOf("error") !== -1) {
                    // Success in upload
                    console.log("Upload complete");
                    setTimeout(fetchPage, 2000);
                } else {
                    // Error in upload
                    console.log("Upload error");
                }
                _this.uploading(false);
                _this.uploadBox(false);
            });
            return true;
        },

        nextPage: function() {
            if (this.page() >= this.total_pages()) return;
            this.page(this.page()+1);
            this.fetchPage();
        },

        previousPage: function() {
            if (this.page() <= 1) return;
            this.page(this.page()-1);
            this.fetchPage();
        },

        fetchPage: function() {
            var page_no = this.page();
            console.log("Fetching page: " + page_no);
            var _this = this;
            $.ajax({
                method: "GET",
                url: "/images/" + page_no,
                success: function(str) {
                    resp = JSON.parse(str);
                    _this.page(resp.page);

                    console.log(resp);

                    if (_this.total_pages() != resp.total_pages) {
                        _this.total_pages(resp.total_pages);
                    }

                    _this.images.removeAll();
                    for(var i in resp.images) {
                        var image = resp.images[i];
                        _this.images.push(image);
                    }
                },
                error: function(e) {
                }
            });
        }
    };

    ko.applyBindings(MainView);

    var username = $.cookie("user");
    if (username != null && username != "") {
        MainView.loggedIn(true);
    }
    MainView.fetchPage();
});