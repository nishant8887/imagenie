$(document).ready(function() {
	var MainView = {
		loggedIn: ko.observable(true),
		signInBox: ko.observable(true),
		uploadBox: ko.observable(false),

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

		image_groups: ko.observableArray([
			{
				a:  { src: "/static/assets/images/picture_1.jpg" },
				b:  { src: "/static/assets/images/picture_2.jpg" },
				c:  { src: "/static/assets/images/picture_3.jpg" }
			}
		]),

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
			this.uploadBox(false);
			return true;
		}
	};

	ko.applyBindings(MainView);
});