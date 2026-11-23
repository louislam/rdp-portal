<template>
  <div class="app">
    <!-- banner rebuilt from the icon plus CSS text, not banner.png -->
    <header class="banner">
      <img class="banner-icon" src="/icon.png" alt=""/>
      <div class="banner-text">
        <div class="banner-line1">Remote Desktop Protocol</div>
        <div class="banner-line2">Portal</div>
      </div>
    </header>

    <main class="main">
      <aside class="list-pane">
        <ul class="list-group">
          <li
              v-for="(profile, i) in profiles"
              :key="i"
              class="list-group-item profile-item"
              :class="{active: i === selected}"
              @click="select(i)"
              @dblclick="connect(i)"
          >
            <img class="profile-icon" src="/icon.png" alt=""/>
            <span class="profile-name">{{ profile.name }}</span>
          </li>
        </ul>
      </aside>

      <section v-if="profiles.length > 0" class="detail-pane">
        <input
            v-model="form.name"
            :disabled="!editMode"
            class="form-control form-control-sm"
            placeholder="Profile name"
        />

        <fieldset class="connection">
          <legend>Connection</legend>

          <div class="field field-host">
            <label class="form-label" for="hostname">Hostname:</label>
            <input id="hostname" v-model="form.hostname" :disabled="!editMode" class="form-control form-control-sm"/>
            <input v-model="form.port" :disabled="!editMode" class="form-control form-control-sm port"
                   placeholder="3389" inputmode="numeric" aria-label="Port"/>
          </div>

          <div class="field">
            <label class="form-label" for="username">User name:</label>
            <input id="username" v-model="form.username" :disabled="!editMode" class="form-control form-control-sm"/>
          </div>

          <div class="field">
            <label class="form-label" for="password">Password:</label>
            <input id="password" v-model="form.password" :disabled="!editMode" type="password"
                   class="form-control form-control-sm"/>
          </div>

          <div class="field">
            <label class="form-label" for="domain">Domain:</label>
            <input id="domain" v-model="form.domain" :disabled="!editMode" class="form-control form-control-sm"
                   placeholder="(Optional)"/>
          </div>
        </fieldset>

        <div class="row-actions">
          <button v-if="!editMode" class="btn btn-outline" @click="startEdit">Edit</button>
          <button v-else class="btn btn-solid theme-primary" @click="save">Save</button>
          <button v-if="editMode" class="btn btn-outline" @click="cancel">Cancel</button>
          <button class="btn btn-outline ms-auto" :disabled="editMode" @click="moreOptions">More RDP Options...</button>
        </div>
      </section>
    </main>

    <footer class="footer">
      <div class="footer-left">
        <button class="btn btn-outline" @click="add">New</button>
        <button v-if="profiles.length > 0" class="btn btn-outline" @click="remove">Delete</button>
      </div>

      <label v-if="profiles.length > 0" class="keep-opening">
        <input type="checkbox" v-model="keepOpening" @change="saveKeepOpening"/>
        Keep opening RDP Portal
      </label>

      <div class="footer-right">
        <button v-if="profiles.length > 0" class="btn btn-solid theme-primary" @click="connect(selected)">
          Connect
        </button>
        <button class="btn btn-outline" @click="showAbout">About</button>
      </div>
    </footer>

    <div v-if="aboutOpen" class="about-backdrop" @click.self="aboutOpen = false">
      <div class="about-dialog">
        <div class="about-titlebar">
          <span>About</span>
          <button class="about-close" aria-label="Close" @click="aboutOpen = false">✕</button>
        </div>
        <div class="about-body">
          <div class="about-name">RDP Portal v{{ version }}</div>
          <div class="about-author">By Louis Lam</div>
          <a class="about-link" href="#"
             @click.prevent="openLink('https://github.com/louislam/rdp-portal')">https://github.com/louislam/rdp-portal</a>
          <button class="btn btn-outline about-ok" @click="aboutOpen = false">OK</button>
        </div>
      </div>
    </div>

    <div v-if="errorMessage" class="alert" role="alert">
      <span class="alert-text">{{ errorMessage }}</span>
      <button class="btn btn-outline alert-dismiss" @click="errorMessage = ''">Dismiss</button>
    </div>
  </div>
</template>

<script lang="ts">
import {PortalService} from "../bindings/github.com/louislam/rdp-portal";

interface ProfileView {
  index: number;
  name: string;
  hostname: string;
  port: string;
  username: string;
  domain: string;
  password: string;
  filename: string;
  isNew: boolean;
}

// Go returns user-facing messages; strip the JS wrapper Wails adds.
function friendlyError(err: unknown): string {
  let message = err instanceof Error ? err.message : String(err);
  message = message.replace(/^(RuntimeError|Error|Uncaught)\s*:\s*/i, "").trim();
  switch (true) {
    case message === "save the profile first":
      return "Save this profile first.";
    case message === "hostname is required":
      return "Hostname is required.";
    case message === "name is required":
      return "Name is required.";
    case message.startsWith("invalid port"):
      return "Port must be a number between 1 and 65535.";
    case message.startsWith("mstsc.exe failed"):
      return "Could not start mstsc.exe.";
    default:
      return message.charAt(0).toUpperCase() + message.slice(1) + (message.endsWith(".") ? "" : ".");
  }
}

export default {
  name: "App",
  data() {
    return {
      profiles: [] as ProfileView[],
      selected: 0,
      editMode: false,
      form: {name: "", hostname: "", port: "", username: "", password: "", domain: ""},
      keepOpening: true,
      version: "",
      aboutOpen: false,
      errorMessage: "",
    };
  },
  async mounted() {
    this.version = await PortalService.AppVersion();
    this.keepOpening = await PortalService.KeepOpening();
    await this.reload();
  },
  methods: {
    // Reload the list from Go (the Go side owns the state) and keep the selection.
    async reload() {
      this.profiles = await PortalService.List();
      if (this.profiles.length === 0) {
        return;
      }
      if (this.selected >= this.profiles.length) {
        this.selected = this.profiles.length - 1;
      }
      this.fillForm(this.profiles[this.selected]);
    },

    fillForm(profile: ProfileView) {
      this.form = {
        name: profile.isNew ? "" : profile.name,
        hostname: profile.hostname,
        port: profile.port,
        username: profile.username,
        password: profile.password,
        domain: profile.domain,
      };
      this.editMode = profile.isNew;
    },

    select(index: number) {
      this.selected = index;
      this.fillForm(this.profiles[index]);
    },

    startEdit() {
      this.editMode = true;
    },

    async add() {
      await PortalService.Add();
      await this.reload();
      this.selected = this.profiles.length - 1;
      this.fillForm(this.profiles[this.selected]);
    },

    async save() {
      try {
        await PortalService.Save(
            this.selected,
            this.form.name,
            this.form.hostname,
            this.form.port,
            this.form.username,
            this.form.password,
            this.form.domain,
        );
        this.editMode = false;
        await this.reload();
        this.selected = Math.min(this.selected, this.profiles.length - 1);
        if (this.profiles[this.selected]) {
          this.fillForm(this.profiles[this.selected]);
        }
      } catch (err) {
        this.errorMessage = friendlyError(err);
      }
    },

    async cancel() {
      const profile = this.profiles[this.selected];
      if (profile && profile.isNew) {
        await PortalService.Discard(this.selected);
      }
      this.editMode = false;
      await this.reload();
    },

    async remove() {
      if (!confirm("Are you sure to delete this profile?")) {
        return;
      }
      await PortalService.Remove(this.selected);
      this.selected = Math.max(0, this.selected - 1);
      await this.reload();
    },

    async connect(index: number) {
      try {
        await PortalService.Connect(index);
      } catch (err) {
        this.errorMessage = friendlyError(err);
      }
    },

    async moreOptions() {
      try {
        await PortalService.Options(this.selected);
      } catch (err) {
        this.errorMessage = friendlyError(err);
      }
    },

    async saveKeepOpening() {
      await PortalService.SetKeepOpening(this.keepOpening);
    },

    async showAbout() {
      this.aboutOpen = true;
    },

    openLink(url: string) {
      window.open(url, "_blank");
    },
  },
};
</script>








