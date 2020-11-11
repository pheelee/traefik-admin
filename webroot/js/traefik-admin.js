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

var app = new Vue({
    el: '#app',
    data: {
      message: 'Proxy Connections',
      connections: [],
      filter_view:[],
      filter_string: '',
      modal_errors: {},
      editor: {
        name: '',
        domain: '',
        backend: '',
      },
      editorMode: 'Create',
    },
    methods: {
        send: function(senderId){
            var method = 'POST'
            if(app.editorMode === 'Update'){
              method = 'PUT'
            }
            ajax('config/'+app.editor.name,method,app.editor, function(){
                if (app.editorMode === 'Create') 
                  app.connections.push(app.editor)
                if (app.editorMode === 'Update')
                  app.connections[app.connections.findIndex(el => el.name === app.editor.name)] = Object.assign({},app.editor)
                M.Modal.getInstance(document.getElementById(senderId)).close()
            }, function(response){
              app.modal_errors = JSON.parse(response).Field
            })
        },
        remove: function(event){
          var id = event.target.dataset["id"]
          var name = app.connections[id].name
          ajax('config/' + name, 'DELETE', null, function(){
            app.connections.splice(id, 1)
          })

        },
        edit: function(){
          var id = event.target.dataset["id"]
          app.editorMode = 'Update'
          app.editor = Object.assign({},app.connections[id])
          M.Modal.getInstance(document.getElementById('editModal')).open();
        },
        applyFilter: function(){
          let filter = app.filter_string.toLowerCase()
          app.filter_view = app.connections.filter(c => 
            c.domain.toLowerCase().includes(filter) || c.name.toLowerCase().includes(filter) || c.backend.toLowerCase().includes(filter)
            )
        }
    }
  })


  document.addEventListener('DOMContentLoaded', function() {

    M.Modal.init(document.querySelectorAll('.modal'), {
      onCloseEnd: function(el) {
        app.editor = {}
        app.editorMode = 'Create'
        app.modal_errors = []
        el.querySelectorAll("input").forEach((i) => {
          i.classList.remove("valid")
        })
      }
    });
    M.Tabs.init(document.querySelectorAll(".tabs"), null);

    ajax('config/','GET',null, function(data){
        app.connections = JSON.parse(data)
        app.filter_view = app.connections
        document.getElementById("connectionList").style.display = "block"
    });

    
  });