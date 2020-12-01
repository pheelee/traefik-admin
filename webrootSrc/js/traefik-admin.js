function ajax(url, method, data, success, failure) {
  var xhr = window.XMLHttpRequest ? new XMLHttpRequest() : new ActiveXObject("Microsoft.XMLHTTP");
  xhr.open(method, url);
  xhr.onreadystatechange = function() {
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
    name: '',
    domain: '',
    backend: '',
    https: true,
    forcetls: true,
    hsts: true,
    headers: [],
    basicauth: [],
    ipRestriction: {depth: 0, ips: []}
  },
  modal_errors: {Field: {},generic:[]},
  structs: {
    headers: {name:'',value:''},
    basicauth: {Username: '',password:''}
  }
}

var app = new Vue({
    el: '#app',
    data: {
      message: 'Proxy Connections',
      connections: [],
      filter_view:[],
      filter_string: '',
      modal_errors: Object.assign({}, defaults.modal_errors),
      editor: Object.assign({},defaults.editor),
      editorMode: 'Create',
    },
    methods: {
        send: function(senderId){
          let tabs = document.querySelector("#editModal .tabs");
          var firstId = tabs.querySelectorAll("a")[0].href.split("#")[1];
          (M.Tabs.getInstance(tabs)).select(firstId);
          if(app.editor.name === '') {app.editor.name='T'}
            var method = 'POST';
            if(app.editorMode === 'Update'){
              method = 'PUT';
            }
            ajax('config/'+app.editor.name,method,app.editor, function(data){
                let config = JSON.parse(data)
                if (app.editorMode === 'Create') 
                  app.connections.push(config);
                if (app.editorMode === 'Update')
                  app.connections[app.connections.findIndex(el => el.name === app.editor.name)] = config;
                M.Modal.getInstance(document.getElementById(senderId)).close();
            }, function(response){
              app.modal_errors = JSON.parse(response);
            })
        },
        remove: function(event){
          var id = event.target.dataset["id"];
          var name = app.connections[id].name;
          ajax('config/' + name, 'DELETE', null, function(){
            app.connections.splice(id, 1);
          })

        },
        edit: function(){
          var id = event.target.dataset["id"];
          app.editorMode = 'Update';
          app.editor = Object.assign({},app.connections[id]);
          M.Modal.getInstance(document.getElementById('editModal')).open();
        },
        applyFilter: function(){
          let filter = app.filter_string.toLowerCase();
          app.filter_view = app.connections.filter(c => 
            c.domain.toLowerCase().includes(filter) || c.name.toLowerCase().includes(filter) || c.backend.toLowerCase().includes(filter)
            );
        },
        addHeader: function(){
          app.editor.headers.push(Object.assign({}, defaults.structs.headers));
        },
        addBasicAuth: function(){
          app.editor.basicauth.push(Object.assign({}, defaults.structs.basicauth));
        },
        addIPRestriction: function(){
          app.editor.ipRestriction.ips.push("")
        }
    }
  })


  document.addEventListener('DOMContentLoaded', function() {

    M.Modal.init(document.querySelectorAll('.modal'), {
      onCloseEnd: function(el) {
        app.editor = Object.assign({},defaults.editor);
        app.editorMode = 'Create';
        app.modal_errors = Object.assign({}, defaults.modal_errors);
        el.querySelectorAll("input").forEach((i) => {
          i.classList.remove("valid");
        })
        el.querySelectorAll("label").forEach((i) => {
          i.classList.remove("active");
        })
      },
      onOpenEnd: function(el) {
        let tabs = el.querySelector(".tabs");
        var firstId = tabs.querySelectorAll("a")[0].href.split("#")[1];
        (M.Tabs.getInstance(tabs)).select(firstId);
      }
    });
    M.Tabs.init(document.querySelectorAll(".tabs"), {});
    ajax('config/','GET',null, function(data){
        app.connections = JSON.parse(data);
        app.filter_view = app.connections;
        document.getElementById("connectionList").style.display = "block";
    });

    
  });