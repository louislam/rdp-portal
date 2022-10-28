using System;
using System.Collections.Generic;
using System.IO;
using System.Windows.Forms;
using Newtonsoft.Json;

namespace RDP_Portal {
    public class Profile {
        private string _name = "";

        public string Name {
            get {
                if (_name == "") {
                    return "<New Profile>";
                }
                return _name;
            }
            set => _name = value;
        }

        public string Filename { get; set; } = "";
        public string Computer { get; set; }
        public string Username { get; set; }

        /**
         * Encrypted Password used by mstsc.exe
         */
        public string GetRDPEncryptedPassword() {
            var mstscpw = new Mstscpw();
            return mstscpw.encryptpw(this.Password);
        }

        /**
         * Encrypted Password in config.json
         */
        public string EncryptedPassword { get; set; } = "";

        [JsonIgnore]
        public string Password {
            get {
                if (EncryptedPassword == "") {
                    return EncryptedPassword;
                }
                return EncryptedPassword.Decrypt();
            }
            set => EncryptedPassword = value.Encrypt();
        }

        public string Domain { get; set; }

        public void PrepareRdpFile() {
            var justCreated = false;

            if (Filename == null || Filename == "") {
                String name;
                while (true) {
                    name = Config.rdpDir + "\\" + StringUtil.GenerateName(8) + ".rdp";
                    if (!File.Exists(name)) {
                        var file = File.Create(name);
                        file.Close();
                        justCreated = true;
                        break;
                    }
                }
                Filename = name;
            }

            if (!File.Exists(Filename)) {
                var file = File.Create(Filename);
                file.Close();
                justCreated = true;
            }

            var lines = File.ReadAllLines(Filename);
            var removeList = new [] {
                "full address:",
                "username:",
                "password",
                "domain:",
            };

            var result = new List<string>();

            foreach (var line in lines) {
                var ok = true;

                foreach (var startKeyword in removeList) {
                    if (line.StartsWith(startKeyword)) {
                        ok = false;
                        break;
                    }
                }

                if (ok) {
                    result.Add(line);
                }
            }

            if (Computer != "") {
                result.Add("full address:s:" + Computer);
            }

            if (Username != "") {
                result.Add("username:s:" + Username);
            }

            if (Password != "") {
                result.Add("password 51:b:" + GetRDPEncryptedPassword());
            }

            if (Domain != "") {
                result.Add("domain:s:" + Domain);
            }

            if (justCreated) {
                result.Add("authentication level:i:0");
                result.Add("prompt for credentials:i:0");
            }

            var writer = new StreamWriter(Filename, false);

            foreach (var line in result) {
                writer.WriteLine(line);
            }

            writer.Close();
        }

        [JsonIgnore] public bool JustAdded { get; set; } = false;

        public void Delete() {
            try {
                File.Delete(Filename);
            } catch (Exception ex) {

            }
        }
    }
}
