<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
	#addDevicePage .tipText{
		padding-bottom:20px;
	}
</style>
<%-- 窗口-添加、修改设备组 --%>
<%-- <div id="winAddOrModDeviceGroup" style="width: 100%;height: 100%;">
	<div class="easyui-layout" data-options="border:false,fit:true">
		<div region="center" data-options="border:false" style="padding: 20px;">
			<div class="itemDiv" style="height: 45px;">
				<span ><%=rb.getString("SheBeiZuMingCheng")%></span>
				<input type="text" name="deviceGroup_name" onblur="validateByRegex(event);" 
					must="1" vali-regex="/^[a-zA-Z0-9_\u4e00-\u9fa5]{1,50}$/" 
					err_prompt_id="deviceGroupNameFmtPrompt"
					class="border border-box item">
				<span style="display: inline-block;padding-left: 185px;">
					<span id="deviceGroupNameFmtPrompt" style="display:none;color:#E03030;"><%=rb.getString("ZiMuShuZiXiaHuaXianHanZi")%></span>	
				</span>			
			</div>
			<div class="itemDiv" style="margin-top: 20px;">
				<span style="vertical-align: top"><%=rb.getString("MiaoShu")%></span>
				<textarea rows="4" cols="20" name="deviceGroup_desc" maxlength=200 class="border border-box item" style="width: 250px; height: 100px; resize: none;"></textarea>
			</div>
		</div>
		<div region="south" data-options="border:false,height:57" >
			<div class="windowButtonGroup" style="margin-right:24px;">
				<a href="#" class="linkbutton linkbutton_trend" onclick="saveDeviceGroup()"><span><%=rb.getString("QueDing")%></span></a>
				<a href="#" class="linkbutton linkbutton_nowanna" onclick="closeDefaultWindow();"><span><%=rb.getString("QuXiao")%></span></a>
			</div>
		</div>
	</div>
</div> --%>
<div id="addDevicePage">
	<el-form  :model='enodebForm' :rules="enodebRules" ref="enodebFormadd" label-position="top">
		<p class="tipText"><%=rb.getString("eNBZhuCeTiShiWenZi")%></p>
		<el-form-item label='<%=rb.getString("XiaoZhanBianMa")%>' prop='serialnumber'>
			<el-input type="textarea" v-model="enodebForm.serialnumber"></el-input>
		</el-form-item>
		<el-form-item label='<%=rb.getString("SheBeiZuMingCheng")%>'>
			<el-select v-model="enodebForm.groupId" >
				<el-option v-for="item in deviceGroupOptions" :key="item.id" :label="item.group_name" :value="item.id"></el-option>
			</el-select>
		</el-form-item>
				
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
				var serialNumber = value,
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
				
				var temp = /^(\d|[a-zA-Z]|-|\s){1,30}$/;
			    if (serialNumber == null || serialNumber.length == 0) {
					callback(new Error('<%=rb.getString("SNBuNengWeiKong")%>'));
				}else{
					var nameFlag = list.every(function(item,index){
						return temp.test(item)
					})
					if(nameFlag){
						callback()
					}else{
						callback(new Error('<%=rb.getString("QingShuRuZhengQueSn")%>'));
					}
					
				}
			};
		return {
			height:'200px',
			enodebForm:{
				serialnumber:'',
				groupId:'',
			},
			enodebRules:{
				serialnumber:[
					{validator:validatorNum,trigger:'blur'}
				]
			},
			type:'',
			deviceGroupOptions:'',
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
			var param={}  , vm = this , url ,id;
			var message = "<%=rb.getString("TianJiaSheBeiChengGong")%>";
				id = vm.enodebForm.groupId;
				url = "${ctx}/system/deviceGroup/addAndAssignEnb.action";
				vm.$refs["enodebFormadd"].validate((valid) => {
					if(valid){
						var snStr = vm.enodebForm.serialnumber||'';

						snStr = snStr.replace(/[(;\s*)(\r\n)\r\n]+/g,';');
						
						axios.post(url,stringify({
							"group_id":id,
							"serialNumber": snStr
						})).then(function(response){
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
<%-- 

// 添加基站
function addDevice() {
	$("#addDevice input.item").blur();
	if ($("#addDevice input.item.err_border").length > 0) {
		$("#addDevice input.item.err_border").fadeOut().fadeIn();
		return;
	}
	
	var selGroup = $("#gridDeviceGroup").datagrid("getSelected");
	var serialNumber = $("#addDevice input[name='serialNumber']").val();
	var temp = /^(\d|[a-zA-Z]|-){1,30}$/;
	if (serialNumber != null && serialNumber.length != 0 && !temp.test(serialNumber)) {
		showMsg('prompt_msg','<%=rb.getString("QingShuRuZhengQueSn")%>');
		return;
	}
	if (serialNumber == null || serialNumber.length == 0) {
		showMsg('prompt_msg','<%=rb.getString("SNBuNengWeiKong")%>');
		return;
	}
	var params = {
		"group_id": selGroup["id"],
		"serialNumber": serialNumber
    };
	$.post("${ctx}/system/deviceGroup/addDevice.action", params, function(data) {
		if (data["success"]) {
			$("#tableDeviceGroupCell").datagrid("reload");
			closeWinAddDevice();
			showMsg('success_msg',data["message"]);
		} else {
			showMsg('error_msg',data["message"]);
		}
	}, "json");
} --%>