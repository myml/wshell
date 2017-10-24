$(document).ready(function(){
    $('select').material_select()
    $('.modal').modal()
    setTimeout(function () {
        $('#sshList').modal('open')
    },100)
});

var sshSturct={
    ID:0,
    Host:"",
    User:"",
    Title:"",
    AuthType:"",
    Key:"",
    Pwd:""
}
$.get("user",function (result) {
    if(result.Err==null){
        sshSturct.User=result.Result.Username
        sshSturct.AuthType="key"
        sshSturct.Key=result.Result.HomeDir+"/.ssh/id_rsa"
    }
})

var sshMain=new Vue({
    el:"#sshMain",
    data:{
        sList:[],
        wsList:[],
        termList:[],
        showIndex:0
    },
    methods:{
        open:function(s){
            this.showIndex=this.sList.push(s)-1
            Vue.nextTick(function () {
                var term = new Terminal()
                term.open(document.getElementById('t' + s.ID))
                sshMain.termList.push(term)
                var ws=new WebSocket("ws://127.0.0.1:4000/ssh/"+s.ID)
                sshMain.wsList.push(ws)
                ws.onopen=function () {
                    term.on("data",function (data) {
                        ws.send(JSON.stringify({
                            Type:"data",
                            Data:data,
                        }))
                    })
                    term.on("resize",function (size) {
                       console.log(size)
                       ws.send(JSON.stringify({
                           Type:"resize",
                           Cols:size.cols,
                           Rows:size.rows
                       }))
                    })
                    term.fit()
                    ws.onmessage=function (msg) {
                       var data=msg.data
                       term.write(data)
                    }
                    ws.onerror=function (e) {
                       console.log(e)
                    }
                    ws.onclose=function (e) {
                       console.log(e)
                    }
                }
            })
        },
        resize:function () {
            this.termList[this.showIndex].fit()
        },
        select:function (i) {
            this.showIndex=i
            Vue.nextTick(function () {
                sshMain.resize()
                sshMain.termList[sshMain.showIndex].focus()
            })
        }
    },
})


var term = new Terminal()
term.open(document.getElementById('z'))
term.fit()
term.on("data",function (data) {
    if(data=="\r") {
        term.reset()
    }else{
        term.write(data)
    }
    for(var i in sshMain.sList){
        sshMain.wsList[i].send(JSON.stringify({
            Type:"data",
            Data:data,
        }))
    }
})

var sshList=new Vue({
    el:"#sshList",
    data:{
        sList:[],
    },
    methods:{
        create:function () {
            sshNew.ssh=$.extend({},sshSturct)
            Vue.nextTick(function () {
                $("select").val(sshSturct.AuthType)
                $('select').material_select()
                Materialize.updateTextFields()
                $("#sshNew").modal("open")
            })
        },
        edit:function (s) {
            sshNew.ssh=s
            Vue.nextTick(function () {
                $('select').material_select()
                Materialize.updateTextFields()
                $("#sshNew").modal("open")
            })
        },
        open:function (s) {
            sshMain.open(s)
            $("#sshList").modal("close")
        },
        ref:function () {
            var vue=this
            $.get("ssh",function (result) {
                vue.sList=result["Result"]
            })
        },
    },
    created:function () {
        this.ref()
    }
})

var sshNew=new Vue({
    el:"#sshNew",
    data:{
        ssh:$.extend({},sshSturct)
    },
    methods:{
        authChange:function (el) {
            this.ssh.AuthType=$(el).val()
        },
        ok:function () {
            $.ajax({
                url:"ssh",
                type:this.ssh.ID==0?"POST":"PUT",
                data:JSON.stringify(this.ssh),
                success:function (result) {
                    if(result["Err"]==null){
                        console.log("成功",result)
                        sshList.ref()
                        $("#sshNew").modal("close")
                    }
                }
            })
        },
        copy:function () {
            this.ssh.ID=0
            this.ok()
        },
        del:function () {
                $.ajax({
                    url:"ssh/"+this.ssh.ID,
                    type:"DELETE",
                    success:function (result) {
                        if(result["Err"]==null) {
                            console.log("成功", result)
                            sshList.ref()
                            $("#sshNew").modal("close")
                        }
                    }
                })
        }
    },
    computed:{
        verify:function () {
            return this.ssh.Host!=""&&this.ssh.User!=""&&(this.ssh.AuthType=="key"?this.ssh.Key!="":this.ssh.Pwd!="")
        }
    },
})
