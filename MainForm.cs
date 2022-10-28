using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Data;
using System.Diagnostics;
using System.Drawing;
using System.Linq;
using System.Text;
using System.Threading.Tasks;
using System.Windows.Forms;

namespace RDP_Portal {
    public partial class MainForm : Form {

        private Config _config;
        private bool _editMode = false;
        private Profile selectedProfile = null;

        public MainForm() {
            InitializeComponent();
            _config = Config.GetConfig();
        }
        
        private void MainForm_Load(object sender, EventArgs e) {
            listBox.DataSource = _config.Profiles;

            if (_config.Profiles.Count == 0) {
                AddNewProfile();
            }

            checkBoxKeepOpening.Checked = _config.KeepOpening;
        }

        public bool EditMode {
            get => _editMode;
            set {
                buttonEdit.Visible = !value;
                buttonSave.Visible = value;
                buttonCancel.Visible = value;
                buttonOptions.Enabled = !value;

                buttonConnect.Enabled = !value;

                textBoxName.Enabled = value;
                textBoxComputer.Enabled = value;
                textBoxUsername.Enabled = value;
                textBoxPassword.Enabled = value;
                textBoxDomain.Enabled = value;
            }
        }

        private void AddNewProfile() {
            var profile = new Profile();
            profile.JustAdded = true;
            _config.Profiles.Add(profile);
            listBox.SelectedIndex = _config.Profiles.Count - 1;
        }

        private void buttonMoreOptions_Click(object sender, EventArgs e) {
            ProcessStartInfo startInfo = new ProcessStartInfo {
                CreateNoWindow = false,
                UseShellExecute = false,
                FileName = "mstsc.exe",
                Arguments = "/edit " + GetSelectedProfile().Filename,
            };

            try {
                var exeProcess = Process.Start(startInfo) ?? throw new InvalidOperationException();
                exeProcess.WaitForExit();
            } catch (Exception ex) {
                MessageBox.Show(ex.ToString());
            }
        }


        private void buttonConnect_Click(object sender, EventArgs e) {
            GetSelectedProfile().PrepareRdpFile();

            ProcessStartInfo startInfo = new ProcessStartInfo {
                CreateNoWindow = false,
                UseShellExecute = false,
                FileName = "mstsc.exe",
                Arguments = GetSelectedProfile().Filename,
            };

            try {
                var exeProcess = Process.Start(startInfo) ?? throw new InvalidOperationException();
                exeProcess.WaitForExit();
            } catch (Exception ex) {
                MessageBox.Show(ex.ToString());
            }
        }

        private void listBox_SelectedValueChanged(object sender, EventArgs e) {
            SelectProfile();
        }

        private Profile GetSelectedProfile() {
            return (Profile) listBox.SelectedItem;
        }

        private void SelectProfile(bool force = false) {
            var profile = (Profile) listBox.SelectedItem;

            // Avoid click empty area reset value
            if (profile == selectedProfile && !force) {
                return;
            }

            selectedProfile = profile;

            EditMode = profile.JustAdded;

            textBoxName.Text = profile.Name ;
            textBoxComputer.Text = profile.Computer;
            textBoxUsername.Text = profile.Username ;
            textBoxPassword.Text = profile.Password;
            textBoxDomain.Text = profile.Domain;
        }

        private void buttonEdit_Click(object sender, EventArgs e) {
            EditMode = true;
        }

        private void buttonCancel_Click(object sender, EventArgs e) {
            EditMode = false;

            var profile = GetSelectedProfile();

            if (profile.JustAdded && _config.Profiles.Count > 1) {
                buttonDelete_Click(null, null);
            } else {
                SelectProfile(true);
            }
        }

        private void buttonNew_Click(object sender, EventArgs e) {
            AddNewProfile();
        }

        private void buttonDelete_Click(object sender, EventArgs e) {
            var selectedItems = (Profile) listBox.SelectedItem;
            selectedItems.Delete();
            _config.Profiles.Remove(selectedItems);
            _config.Save();
            if (_config.Profiles.Count == 0) {
                AddNewProfile();
                SelectProfile(true);
            }
        }

        private void buttonSave_Click(object sender, EventArgs e) {
            var profile = (Profile) listBox.SelectedItem;

            profile.JustAdded = false;

            profile.Name = textBoxName.Text;
            profile.Computer = textBoxComputer.Text;
            profile.Username = textBoxUsername.Text;
            profile.Password = textBoxPassword.Text;
            profile.Domain = textBoxDomain.Text;

            profile.PrepareRdpFile();

            _config.Save();
            EditMode = false;

            // Refresh the list
            listBox.DisplayMember = null;
            listBox.DisplayMember = "Name";
        }

        private void checkBoxKeepOpening_CheckedChanged(object sender, EventArgs e) {
            _config.KeepOpening = checkBoxKeepOpening.Checked;
            _config.Save();
        }
    }
}
