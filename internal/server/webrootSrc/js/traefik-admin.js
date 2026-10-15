function ajax(url, method, data, success, failure, progress=true) {
  if (progress) Loader.Show();
  var xhr = window.XMLHttpRequest ? new XMLHttpRequest() : new ActiveXObject("Microsoft.XMLHTTP");
  xhr.open(method, url);
  xhr.onreadystatechange = function() {
      if (progress) Loader.Hide();
      if (xhr.readyState>3 && xhr.status==200) { success(xhr.responseText); }
      if (xhr.readyState>3 && xhr.status > 399) {failure(xhr.responseText); }
  };
  xhr.setRequestHeader('X-Requested-With', 'XMLHttpRequest');
  xhr.setRequestHeader('Content-Type', 'application/json');
  xhr.send(JSON.stringify(data));
  return xhr;
}

var defaults = {
  editor: {
    id: '',
    name: '',
    domain: '',
    backend: {url: '', healthy: true},
    forwardauth: false,
    https: true,
    forcetls: true,
    hsts: true,
    headers: [
      {name:'',value:''},
      {name:'',value:''},
      {name:'',value:''},
      {name:'',value:''},
      {name:'',value:''},
    ],
    basicauth: [
      {Username: '', password:''},
      {Username: '', password:''},
      {Username: '', password:''},
      {Username: '', password:''},
      {Username: '', password:''},
    ],
    ipRestriction: {depth: 0, ips: ["","","","",""]},
  },
  validation: {
    valid: true,
    errors: {
      name: '',
      domain: '',
      backend: '',
      basicauth: [
        {username:'',password:''},
        {username:'',password:''},
        {username:'',password:''},
        {username:'',password:''},
        {username:'',password:''},
      ],
      allowedip: {
        noproxies: '',
        ip: ['','','','','']
      },
      headers: [
        {name:'',value:''},
        {name:'',value:''},
        {name:'',value:''},
        {name:'',value:''},
        {name:'',value:''},
      ]
    }
  }
}

function openDialog(id) {
  const dialog = document.getElementById(id);
  dialog.showModal();
  const tabs = dialog.querySelector('.tabs');
  if (tabs) {
    const firstId = tabs.querySelector('a').hash.slice(1);
    M.Tabs.getInstance(tabs).select(firstId);
  }
}

var app = Vue.createApp({
    render: TraefikAdminRender,
    data: function() { return {
      features: {
        forwardauth: {
          enabled: false,
          url: ''
        },
        version: 'dev'
      },
      copyright: (new Date()).getFullYear() + ' Philipp Ritter',
      confirmDialog: {
        title: 'Confirm', text: '', id: 0,
        onYes: function() {},
        onNo: function() {},
      },
      endpoint: '',
      message: 'Proxy Connections',
      connections: [],
      filter_view:[],
      filter_string: '',
      validation: JSON.parse(JSON.stringify(defaults.validation)),
      editor: JSON.parse(JSON.stringify(defaults.editor)),
      editorMode: 'Create',
    } },
    methods: {
        openEditor: function() {
          openDialog('editModal');
        },
        send: function(senderId){
          let tabs = document.querySelector("#editModal .tabs");
          var firstId = tabs.querySelectorAll("a")[0].href.split("#")[1];
          (M.Tabs.getInstance(tabs)).select(firstId);
            var method = 'POST';
            if(app.editorMode === 'Update'){
              method = 'PUT';
            }
            var configId = method === 'POST' ? 'new' : app.editor.id;
            ajax('config/'+encodeURIComponent(configId),method,app.editor, function(data){
                let config = JSON.parse(data)
                if (app.editorMode === 'Create') 
                  app.connections.push(config);
                if (app.editorMode === 'Update')
                  app.connections[app.connections.findIndex(el => el.id === app.editor.id)] = config;
                document.getElementById(senderId).close();
                app.applyFilter();
                Notify.Success(app.editor.name, "successfully " + app.editorMode.toLowerCase() + "d")
            }, function(response){
              app.validation = JSON.parse(response);
              Notify.Error(app.editor.name, "failed to create/update")
            })
        },
        remove: function(event){
          var id = event.target.dataset["id"];
          var c = app.connections.filter(e=>e.id == id)[0]
          var name = c.name;
          app.confirmDialog.title = 'Delete Config';
          app.confirmDialog.text  = "Do you really want to delete " + name + " ?";
          app.confirmDialog.id    = c.id;
          app.confirmDialog.onYes = function(){
            ajax('config/' + c.id, 'DELETE', null, function(){
              app.connections.splice(app.connections.findIndex(e => e.id == app.confirmDialog.id), 1);
              app.applyFilter();
              Notify.Success(name, "config deleted")
            })
          };
          openDialog('confirmModal');
        },
        edit: function(event){
          var id = event.target.dataset["id"];
          app.editorMode = 'Update';
          app.editor = Object.assign({},app.filter_view[id]);
          openDialog('editModal');
        },
        applyFilter: function(){
          let filter = app.filter_string.toLowerCase();
          app.filter_view = app.connections.filter(c => 
            c.domain.toLowerCase().includes(filter) || c.name.toLowerCase().includes(filter) || c.backend.url.toLowerCase().includes(filter)
            );
        },
    }
  }).mount('#app')


  var Notify = {
    Success: function(sender,msg){
      this._fire(sender,msg,"green")
    },
    Error: function(sender,msg){
      this._fire(sender,msg,"red")
    },
    Info: function(sender,msg){
      this._fire(sender,msg)
    },
    _fire: function(sender, msg, color=''){
      new M.Toast({text: sender + ' ' + msg, classes:color})
    }
  }

  var Loader = {
    el: document.getElementById("loaderProgress"),
    Show: function(){
      this.el.style.display = "block"
    },
    Hide: function(){
      this.el.style.display = "none"
    }
  }

  document.addEventListener('DOMContentLoaded', function() {

    document.getElementById('editModal').addEventListener('close', function(event) {
        const el = event.currentTarget;
        app.editor = JSON.parse(JSON.stringify(defaults.editor));
        app.editorMode = 'Create';
        app.validation = JSON.parse(JSON.stringify(defaults.validation));
        el.querySelectorAll("input").forEach((i) => {
          i.classList.remove("valid");
        })
        el.querySelectorAll("label").forEach((i) => {
          i.classList.remove("active");
        })
    });
    document.addEventListener('click', function(event) {
      const close = event.target.closest('.modal-close');
      if (close) {
        event.preventDefault();
        close.closest('dialog').close();
      }
    });
    document.querySelectorAll('dialog.modal').forEach(function(dialog) {
      dialog.addEventListener('click', function(event) {
        const bounds = dialog.getBoundingClientRect();
        if (event.target === dialog &&
            (event.clientX < bounds.left || event.clientX > bounds.right ||
             event.clientY < bounds.top || event.clientY > bounds.bottom)) {
          dialog.close();
        }
      });
    });
    M.Tabs.init(document.querySelectorAll(".tabs"), {});
    ajax('config/','GET',null, function(data){
        app.connections = JSON.parse(data);
        app.filter_view = app.connections;
        document.getElementById("connectionList").style.display = "block";
    });
    ajax('features', 'GET', null, function(data){
      app.features = JSON.parse(data);
    }, function(){}, false)    
  });
