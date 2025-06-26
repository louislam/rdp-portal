using System;
using System.ComponentModel;
using System.IO;
using Newtonsoft.Json;

namespace Core {
    public class Config {

        private static Config _instance;
        
        [JsonIgnore]
        public string ConfigFile;
            
        [JsonIgnore]
        public string RdpDir;
        
        private Config() {
            // Private constructor to prevent instantiation
        }
        
        public static Config GetConfig(string folder) {
            var filename = "config.json";
            
            var configFile = Path.Combine(folder, filename);
            var rdpDir = Path.Combine(folder, "rdp-files");
            
            if (!File.Exists(configFile)) {
                File.AppendAllText(configFile, "{}");
            }

            if (!Directory.Exists(rdpDir)) {
                Directory.CreateDirectory(rdpDir);
            }

            var json = File.ReadAllText(configFile);
            _instance = JsonConvert.DeserializeObject<Config>(json);

            if (_instance == null) {
                throw new Exception("Cannot read " + configFile);
            }
            
            _instance.ConfigFile = configFile;
            _instance.RdpDir = rdpDir;
            
            if (_instance.Profiles == null) {
                _instance.Profiles = new BindingList<Profile>();
                _instance.Save();
            }
            
            // Assign Config to each profile
            foreach (var profile in _instance.Profiles) {
                profile.Config = _instance;
            }

            return _instance;
        }

        public BindingList<Profile> Profiles { get; set; }

        public bool KeepOpening { get; set; } = true;

        public void Save() {
            var json = JsonConvert.SerializeObject(this, Formatting.Indented);
            File.WriteAllText(ConfigFile, json);
        }
    }
}
