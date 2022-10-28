using System;
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

            var writer = File.AppendText(Filename);

            if (Computer != "") {
                writer.WriteLine("full address:s:" + Computer);
            }

            if (Username != "") {
                writer.WriteLine("username:s:" + Username);
            }

            if (Password != "") {
                // TODO
                writer.WriteLine("password 51:b:" + Password);
            }

            if (Domain != "") {
                writer.WriteLine("domain:s:" + Domain);
            }

            if (justCreated) {
                writer.WriteLine("authentication level:i:0");
                writer.WriteLine("prompt for credentials:i:0");
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
