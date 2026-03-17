<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<!DOCTYPE html>
<html>
<head>
</head>
<body>
  <div id="device_add">
	<el-form ref="form" :model="form" :rules="rules" label-position="top">
		<p style="padding-bottom:20px;"><%=rb.getString("eNBZhuCeTiShiWenZi")%></p>
		<el-form-item label='<%=rb.getString("XiaoZhanBianMa")%>' prop='serialnumber'>
			<el-input type="textarea" v-model="form.serialnumber"></el-input>
		</el-form-item>
		<el-form-item label='<%=rb.getString("SheBeiZuMingCheng")%>'>
			<el-select v-model="form.groupId" >
				<el-option v-for="item in deviceGroupOptions" :key="item.id" :label="item.group_name" :value="item.id"></el-option>
			</el-select>
		</el-form-item>
	</el-form>
	<div style="padding: 20px 0 0 0px;">
		<el-button type="primary" @click="addeGnodeb"><%=rb.getString("QueDing")%></el-button>
		<el-button @click="closeAddDevice"><%=rb.getString("QuXiao")%></el-button>
	</div>
  </div>
  <script type="text/javascript">
  	new Vue({
		el: '#device_add',
		data() {
			var vm = this,
				validatorNum = (rule,value,callback) => {
					var serialNumber = value,
						serialNumberArr =  serialNumber.split(";");

					//判断最后一项是否为空 为空删除
					if(serialNumberArr[serialNumberArr.length-1] == ""){
						serialNumberArr.splice(serialNumberArr.length-1)
					}

					var temp = /^(\d|[a-zA-Z]|-|\s){1,30}$/;
					if (serialNumber == null || serialNumber.length == 0) {
						callback(new Error('<%=rb.getString("SNBuNengWeiKong")%>'));
					}else{
						var nameFlag = serialNumberArr.every(function(item,index){
								return temp.test(item)
							});

						if(nameFlag){
							callback()
						}else{
							callback(new Error('<%=rb.getString("QingShuRuZhengQueSn")%>'));
						}
						
					}
				};

			return {
				form: {
					serialnumber: '',
					groupId: ''
				},
				rules:{
					serialnumber:[
						{validator:validatorNum}
					]
				},
				deviceGroupOptions: []
			};
		},
		methods: {
			init(params) {
				var vm = this,
					p = params || {},
					groupId = p.groupId;

				axios.post('${ctx}/system/deviceGroup/getSimpleDeviceGroupList.action',stringify({isAll:'0'})).then(function(response){
					let data = response.data
					vm.deviceGroupOptions = data;
					vm.form.groupId = data[0].id;
				}).catch(function(error){})
			},
			addeGnodeb(){
				var param={}  , vm = this , url ,
					id = vm.form.groupId,
					message = '<%=rb.getString("TianJiaSheBeiChengGong")%>';

				var url = "${ctx}/system/deviceGroup/addAndAssignEnb.action";

				vm.$refs["form"].validate((valid) => {
					if(valid){
						axios.post(url,stringify({
							"group_id": id,
							"serialNumber": vm.form.serialnumber,
							isGnb: 1
						})).then(function(response){
							var data = response.data;
							if (data.success){
								eventBus.$emit('close-gnb-dialog');
								eventBus.$emit('refresh-device-list');
								vm.$message({
									message: message,
									type:'success',
								})
							}else {
								vm.$message.error(data.message)
							}
						}).catch(function(error){})
					}else{}
				})
			},
			closeAddDevice(){
				eventBus.$emit('close-gnb-dialog');
			},
		},
		mounted() {
			var vm = this;

			eventBus.$off('init-gnb');
			eventBus.$on('init-gnb',vm.init);
		}
	})
  </script>
</body>
</html>