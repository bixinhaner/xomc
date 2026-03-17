<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
	#addDevicePage .tipText{
		padding-bottom:20px;
	}
</style>
<%-- 窗口-添加、修改设备组 --%>
<div id="addDevicePage">
	<el-form  :model='enodebForm' :rules="enodebRules" ref="enodebForm" label-position="top">
		<el-form-item  prop='type' label="" style='margin-bottom:10px;'>
			<span style="font-size:14px;color:#333333;margin-right:20px;">Input Type</span>
			<el-radio-group v-model="enodebForm.type">
				<el-radio label="mac" style='margin-right:30px;'>MAC</el-radio>
				<el-radio label="sn" style='margin-bottom:0px;'><%=rb.getString("CPEBianMa")%></el-radio>
			</el-radio-group>
		</el-form-item>
		<el-form-item label='' prop='serialnumber'>
			<el-input type="textarea" v-model="enodebForm.serialnumber"></el-input>
		</el-form-item>
		<p class="tipText"><%=rb.getString("cpeZhuCeTiShiWenZi")%></p>
		<el-form-item label='<%=rb.getString("SheBeiZuMingCheng")%>' prop='groupId'>
			<el-select v-model="enodebForm.groupId" >
				<el-option v-for="item in deviceGroupOptions" :key="item.id" :label="item.group_name" :value="item.id"></el-option>
			</el-select>
		</el-form-item>
		<!-- #48836 删除Link Condition -->
		<%-- <el-form-item label='<%=rb.getString("LianJieTiaoJian")%>'>
			<el-select v-model="enodebForm.link" >
				<el-option v-for="item in deviceLinkOptions" :key="item.value" :label="item.text" :value="item.value"></el-option>
			</el-select>
		</el-form-item> --%>
	</el-form>
	<div>
		<el-button type="primary" @click="addeNodeb"><%=rb.getString("QueDing")%></el-button>
		<el-button @click="closeAddDevice"><%=rb.getString("QuXiao")%></el-button>
	</div>
</div>


<script>

var addDeviceVue = new Vue({
	el:'#addDevicePage',
	data(){
		 var validatorNum = (rule,value,callback) => {
			
			var serialNumber = value||'',
				snArr =  serialNumber.split(/[(\r\n)\r\n]+/g),
				list = [];
			
			if(snArr.length>1) {// 多行
				snArr.map(function(str){
					var item = str.trim(),
						lastIdx = item.lastIndexOf(';'),
						length = item.length-1;
					
					if(lastIdx>=0 && lastIdx == length) {
						list.push(item.substring(0,lastIdx));
					}else if(item) {
						list.push(item);
					}
				});
			}else {// 单行
				list = serialNumber.split(';');
			}
			
			//判断最后一项是否为空 为空删除
			if(list[list.length-1] == ""){
				list.splice(list.length-1)
			}
			
			var temp = /^[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}$/;
			var noColTemp = /^([A-Fa-f0-9]{2}){6}$/;
			var tempSn = /^(\d|[a-zA-Z]|-|\s){1,30}$/;
		    if(serialNumber != null && serialNumber.length != 0){
				var nameFlag;
					if(this.enodebForm.type == 'mac'){
						nameFlag = list.every(function(item,index){
							return (temp.test(item) ||  noColTemp.test(item)) 
						})
					}else{
						nameFlag = list.every(function(item,index){
							return tempSn.test(item)
						})
					}
				if(nameFlag){
					callback()
				}else{
					if(this.enodebForm.type == 'mac'){
						callback(new Error('<%=rb.getString("QingShuRuZhengQueMac")%>'));
					}else{
						callback(new Error('<%=rb.getString("QingShuRuZhengQueSn")%>'));
					}
				}
			}else if (serialNumber == null || serialNumber.length == 0) {
				if(this.enodebForm.type == 'mac'){
					callback(new Error('<%=rb.getString("QingShuRuZhengQueMac")%>'));
				}else{
					callback(new Error('<%=rb.getString("QingShuRuZhengQueSn")%>'));
				}
			}else{
				callback();
			}
		};
		return {
			height:'200px',
			enodebForm:{
				type:'mac',
				serialnumber:'',
				groupId:''
			},
			enodebRules:{
				serialnumber:[
					{validator:validatorNum,trigger:'blur'}
				]
			},
			type:'',
			deviceGroupOptions:'',
			deviceLinkOptions: [
				{text:'NLOS',value:'nlos'},
				{text:'PLOS',value:'plos'},
				{text:'LOS',value:'los'}
			],
			groupId:''
		}
		
	},
	methods:{
		init(groupId){
			var vm = this;
			axios.post('${ctx}/system/deviceGroup/getSimpleDeviceGroupList.action',stringify({isAll:'0'})).then(function(response){
				let data = response.data
				vm.deviceGroupOptions = data;
				vm.enodebForm.groupId = data[0].id;
			}).catch(function(error){})
		},
		closeAddDevice(){
			eventBus.$emit('close-dialog');
		},
		addeNodeb(){
			var vm = this ,
				id = vm.enodebForm.groupId;
				mac = vm.enodebForm.serialnumber || '';
				url = "${ctx}/cell/CPE/addAndAssignCpe.action",
				message = "<%=rb.getString("TianJiaSheBeiChengGong")%>";
				params={
					group_id:vm.enodebForm.groupId,
					type:vm.enodebForm.type
				};
				/* link_condition = vm.enodebForm.link; */
			mac = mac.replace(/[(;\s*)(\r\n)\r\n]+/g,';');
			if(mac) {
				if(vm.enodebForm.type == 'mac'){
					mac = mac.split(';').map(function(item){
						return (item.trim().toLocaleUpperCase().match(/[a-zA-Z0-9]{2}/g) || []).join(':');
					}).join(';');
				}
			}
			
			 vm.$refs.enodebForm.validate((valid) => {
				if(valid){
					var macStr = mac||'';

					macStr = macStr.replace(/[(;\s*)(\r\n)\r\n]+/g,';');
					if(vm.enodebForm.type == 'mac'){
						params.macAddress = macStr;
					}else{
						params.sns = macStr;
					}
					axios.post(url,stringify(params)).then(function(response){
						let data = response.data;
						if (data.success){
							eventBus.$emit("add-enode")
							vm.$message({
								message: message,
								type:'success',
							})
						}else {
							vm.$message.error(data.message)
						}
						
					}).catch(function(error){})
					
				}else{
				}
			}) 
			
		}
	},
	mounted(){
		eventBus.$off('addDevice-dialog').$on('addDevice-dialog',this.init);
	}
	
});

</script>

<%-- <%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

手动添加CPE到设备组
<div id="addCpe" class="easyui-layout" data-options="border:false,fit:true">
    <div region="center" data-options="border:false" style="padding: 10px 30px;" style="margin-top:">
        <div class="itemDiv" style="margin-top:10px;height:50px;">
            <span style="display:block"><%=rb.getString("MACDiZhi")%></span>
            <input style="width:285px;" id="macAddress" type="text" name="macAddress" class="border border-box item" maxlength=30/>
        </div>
        <div id="macAddressTitle" style="color:#797979;margin-top:8px;"><%=rb.getString("QingShuRuZhengQueMac")%></div>
    </div>
    <div region="south" data-options="border:false,height:77">
    	<div class="windowButtonGroup" style="margin-top:20px;margin-right:42px;margin-bottom:10px">
    		<a onclick="addCpe()" class="linkbutton linkbutton_trend" ><span><%=rb.getString("TianJia")%></span></a>
    		<a onclick="closeWinAddCpe()" class="linkbutton linkbutton_nowanna"><span><%=rb.getString("QuXiao")%></span></a>
    	</div>
	</div>
</div>

<script type="text/javascript">
$(function(){
	$("#macAddress").blur(function(){
		var macAddress = $("#addCpe input[name='macAddress']").val();
		macAddress = macAddress.toUpperCase();
	
			var temp = /^[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}$/;
			var noColTemp = /^([A-Fa-f0-9]{2}){6}$/;
			if (!noColTemp.test(macAddress)) {
				if (!temp.test(macAddress)){
					$("#macAddressTitle").css("color","red");
				}else{
					$("#macAddressTitle").css("color","#797979");
				}
		    } 
			if (noColTemp.test(macAddress)){
				$("#macAddressTitle").css("color","#797979");
			}
		
	})
})
// 关闭添加基站窗口
function closeWinAddCpe() {
	/* $("#winAddDevice").window("close"); */
	closeDefaultWindow();
}

// 添加CPE
function addCpe() {
	var selGroup = $("#gridDeviceGroup").datagrid("getSelected");
	var macAddress = $("#addCpe input[name='macAddress']").val();
	macAddress = macAddress.toUpperCase();
	if (macAddress != null && macAddress.length != 0) {
		var temp = /^[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}$/;
		var noColTemp = /^([A-Fa-f0-9]{2}){6}$/;
		if (!noColTemp.test(macAddress)) {
			if (!temp.test(macAddress)){
				$.messager.alert(TiShi, "<%=rb.getString("QingShuRuZhengQueMac")%>");
				$("#macAddressTitle").css("color","red");
		    	return;
			}
	    } 
		if (noColTemp.test(macAddress)){
			//add colmn to mac
			var temp = "" ;
			for(var i=1;i<=6;i++){
				if(i == 6 ){
					 temp += macAddress.substring(2*i-2,2*i);
				} else {
					 temp += macAddress.substring(2*i-2,2*i) + ":";
				}
			}
			macAddress = temp;
		}
	} else {
		$("#macAddressTitle").css("color","red");
		$.messager.alert(TiShi, "<%=rb.getString("MacBuNengWeiKong")%>"); 
    	return;
	}
	var params = {
		"group_id": selGroup["id"],
		"macAddress": macAddress
    };
	$.post("${ctx}/cell/CPE/addCpe.action", params, function(data) {
		if (data["success"]) {
			$("#tableDeviceGroupCpe").datagrid("reload");
			closeWinAddCpe();
			showMsg('success_msg',data["message"]);
		} else {
			showMsg('error_msg',data["message"]);
		}
	}, "json");
}

function validateSn(e) {
	var ele = $(e["target"]);
	var value = ele.val();
	var temp = /^(\d|[a-zA-Z])+$/;
    var currValLength = ele.val().length;
    if (currValLength == 0) {
		$("#" + ele.attr("id") + "_err").show();
		ele.addClass("err_border");
    } else {
    	$("#" + ele.attr("id") + "_err").hide();
    	ele.removeClass("err_border");
    }
    if (!temp.test(value)) {
    	showMsg('prompt_msg',"<%=rb.getString("QingShuRuZhengQueSn")%>");
    }
}

function validateMac(e) {
	var ele = $(e["target"]);
	var value = ele.val();
	var temp = /[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}/;
    var currValLength = ele.val().length;
    if (currValLength == 0) {
		$("#" + ele.attr("id") + "_err").show();
		ele.addClass("err_border");
    } else {
    	$("#" + ele.attr("id") + "_err").hide();
    	ele.removeClass("err_border");
    }
	
    if (!temp.test(value)) {
    	showMsg('prompt_msg',"<%=rb.getString("QingShuRuZhengQueMac")%>");
    }
}
</script> --%>