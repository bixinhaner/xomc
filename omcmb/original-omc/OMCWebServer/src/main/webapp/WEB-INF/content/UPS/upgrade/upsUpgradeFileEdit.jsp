<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
#fileEditUPSDiv .el-form-item__label{
	line-height:26px;
	width:140px;
	text-align:left;
}
#fileEditUPSDiv .el-form-item__error{
	margin-left:140px;
}
#fileEditUPSDiv .el-form-item{
	margin-bottom:35px;
}
#fileEditUPSDiv .el-select .el-input.is-disabled .el-input__inner{
	min-height:26px;
	max-height:26px;
}
</style>

<div id='fileEditUPSDiv'>
	<el-form ref="upsFileEditForm" :model="editForm" :rules="importRules" style="margin-left:30px;margin-top:30px;" :hide-required-asterisk=true>
		<el-form-item prop="product" label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" style='display:inline-block;width:380px;'>
			<el-input v-model='editForm.product' disabled=true></el-input>
		</el-form-item>
		<el-form-item label="<%=rb.getString("WenJianMing")%>" prop="fileName">
			<el-input :disabled="true" v-model="editForm.fileName" style="width:400px;"></el-input>
		</el-form-item>
		<el-form-item label="<%=rb.getString("BanBen")%>" prop="version">
			<el-input v-model='editForm.version' :disabled="viewFlag" style="width:400px;"></el-input>
		</el-form-item>
		
		<el-form-item label='<%=rb.getString("TuiJian")%>' prop='recommend'>
			<el-select v-model='editForm.recommend' :disabled="viewFlag">
				<el-option label='<%=rb.getString("Shi")%>' value='1'></el-option>
				<el-option label='<%=rb.getString("Fou")%>' value='0'></el-option>
			</el-select>
		</el-form-item>
		<el-form-item label="<%=rb.getString("MiaoShu")%>" prop='desc'>
			<el-input :disabled="viewFlag" v-model='editForm.desc' type='textarea' :rows='6' style='width:600px;'></el-input>
		</el-form-item>
	</el-form>
</div>
<script>
	new Vue({
		el:'#fileEditUPSDiv',
		data(){
			var versionValidate = function(rule,value,callback) {
				if(value) {
					if(value.length>45) {
						callback('<%=rb.getString("FanWei")%>:1-45 <%=rb.getString("ZiFuFuShu")%>');
					}else {
						callback();
					}
				}else {
					callback("<%=rb.getString("QingShuRuWenJianBanBen")%>");
				}
			};
			return{
				editForm:{
					product:'UPS',
					fileName:'',
					version:'',
					recommend:'',
					desc:'',
				},
				versionId:'',
				importRules:{
					version:[
						{validator: versionValidate}
					],
					product:[
						{required:true,message:'<%=rb.getString("ShuRuBiTianXiang")%>',trigger:'change'}
					],
					recommend:[
						{required:true,message:'<%=rb.getString("ShuRuBiTianXiang")%>',trigger:'change'}
					]
				},
				viewFlag : false,
			}
		},
		methods:{
			init(id,type){
				var vm = this;
				vm.versionId = id;
				axios.post("${ctx}/cell/version/getDeviceVersionFileInfo.action",stringify({versionId:id})).then(function(response){
						var data = response.data;
						vm.editForm.fileName = data.file_name;
						vm.editForm.version = data.version;
						vm.editForm.recommend =data.recommend;
						vm.editForm.desc = data.desc;
						
						initForm(vm.$refs.upsFileEditForm);
					})
				if(type == 'view'){
					vm.viewFlag = true;
				}
			},
			cancel(){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>'
				if(isFormChanged(vm.$refs.upsFileEditForm)){
					vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						eventBus.$emit('hide-upsUpgrade-slide');
					}).catch(() => {
						
					})
				}else{
					eventBus.$emit('hide-upsUpgrade-slide');
				}
			},
			submit(){
				var vm = this;
				vm.$refs.upsFileEditForm.validate((valid) => {
					if(valid){
						var params={
							versionId:vm.versionId,
							product : vm.editForm.product,
							fileName : vm.editForm.fileName,
							version : vm.editForm.version,
							recommend : vm.editForm.recommend,
							desc : vm.editForm.desc,
                            to_who : 'all'
						}
						axios.post("${ctx}/cell/version/goModifyDeviceVersionFileInfo.action",stringify(params)).then(function(response){
							var data = response.data;
							if(data["success"]){
								vm.$message({
		    						message:"<%=rb.getString("ChengGong")%>",
		    						type:'success',
		    					})
                                eventBus.$emit('hide-upsUpgrade-slide');
							}else{
								vm.$message.error(data["message"])
							}
						})
					}
				})
			}
		},
		mounted(){
			var vm = this;
			eventBus.$off('ups-upgrade-editInt').$on('ups-upgrade-editInt',this.init);
			eventBus.$off('ups-upgrade-editSubmit').$on('ups-upgrade-editSubmit',this.submit);
			eventBus.$off('ups-upgrade-cancelEdit').$on('ups-upgrade-cancelEdit',this.cancel);
		}
	}) 
</script>