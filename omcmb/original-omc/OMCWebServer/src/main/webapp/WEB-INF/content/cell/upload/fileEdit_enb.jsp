<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
#fileEditEnbDiv .el-form-item__label{
	line-height:26px;
	width:140px;
	text-align:left;
}
#fileEditEnbDiv .el-form-item__error{
	margin-left:140px;
}
#fileEditEnbDiv .el-form-item{
	margin-bottom:35px;
}
#fileEditEnbDiv .el-select .el-input.is-disabled .el-input__inner{
	min-height:26px;
	max-height:26px;
}
</style>

<div id='fileEditEnbDiv'>
	<el-form ref="enbFileEditForm" :model="editForm" :rules="importRules" style="margin-left:30px;margin-top:30px;" :hide-required-asterisk=true>
		<el-form-item prop="product" label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" style='display:inline-block;width:380px;' v-if="showProduct" key="product">
			<el-select v-model='editForm.product' :disabled="viewFlag">	
				<el-option v-for='item in productOptions' :label="item.name" :value="item.value"></el-option>
			</el-select>
		</el-form-item>
		<el-form-item label="<%=rb.getString("WenJianMing")%>" prop="fileName">
			<el-input :disabled="true" v-model="editForm.fileName"></el-input>
		</el-form-item>
		<el-form-item label="<%=rb.getString("BanBen")%>" prop="version">
			<el-input v-model='editForm.version' :disabled="viewFlag"></el-input>
		</el-form-item>
		<el-form-item label='<%=rb.getString("Title_KaiFangYunYingShang")%>' prop='to_who' v-show="showToWho">
			<el-select v-model='editForm.to_who' :disabled="viewFlag">
				<el-option label='<%=rb.getString("ShangYongBanBen")%>' value='all'></el-option>
				<el-option label='<%=rb.getString("CeShiBanBen")%>' value='none'></el-option>
				<el-option label="<%=rb.getString("BetaBanBen")%>" value="beta"></el-option>
			</el-select>
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
	var enbFileEditVue = new Vue({
		el:'#fileEditEnbDiv',
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
					product:'',
					fileName:'',
					version:'',
					recommend:'',
					desc:'',
					to_who:'all'
				},
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
				productOptions:[],
				showToWho:false,
				viewFlag : false,
				showProduct:true
			}
		},
		methods:{
			init(){
				var vm = this;
				axios.post("${ctx}/task/upgrade/getProductType.action",stringify({
                			type: 'all'
                		})).then(function(response){
					var data = response.data;
					vm.productOptions = data;
					var id = enbFileVue.rowDataFile.id;
					axios.post("${ctx}/cell/version/getDeviceVersionFileInfo.action",stringify({versionId:id})).then(function(response){
						var data1 = response.data;
						vm.editForm.product = data1.product;
						vm.editForm.fileName = data1.file_name;
						vm.editForm.version = data1.version;
						vm.editForm.recommend =data1.recommend;
						vm.editForm.desc = data1.desc;
						//Cloud版本时，显示可见范围的下拉选项 
						if(isCloudCore == 'true'){
							vm.showToWho = true;
							vm.editForm.to_who = data1.toWho;
						}else{
							vm.showToWho = false;
						}
						var type;
						vm.productOptions.map(function(item){
							if(item.value == data1.product){
								type = item.name;
							}
						})
						initForm(vm.$refs.enbFileEditForm);
					})
				})
				if(enbFileVue.operType == 'viewFile'){
					vm.viewFlag = true;
				}
				if(enbFileVue.file_type == "ap"){
					vm.showProduct = false;
				}
			},
			cancel(){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>'
				if(isFormChanged(vm.$refs.enbFileEditForm)){
					vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						enbFileVue.$refs.slide.hide();
					}).catch(() => {
						
					})
				}else{
					enbFileVue.$refs.slide.hide();
				}
			},
			submit(){
				var vm = this;
				vm.$refs.enbFileEditForm.validate((valid) => {
					if(valid){
						var codes = {
								upgrade:0,
								ca:1,
								fpga:6,
								ap:11
						}
						var nxpObj = {
							"CR-B4860/EU" : '7',
							"CR-B4860/BU" : '10',
							"CR-B4860/RU" : '8',
						}
						var params = {
								versionId : enbFileVue.rowDataFile.id,
								product : vm.editForm.product,
								fileName : vm.editForm.fileName,
								version : vm.editForm.version,
								recommend : vm.editForm.recommend,
								desc : vm.editForm.desc,
								toWho : vm.editForm.to_who
						}
						if(vm.editForm.product.includes("CR-B4860")){
							params.fileType = nxpObj[vm.editForm.product];
						}else{
							params.fileType = codes[enbFileVue.file_type];
						}
						axios.post("${ctx}/cell/version/goModifyDeviceVersionFileInfo.action",stringify(params)).then(function(response){
							var data = response.data;
							if(data["success"]){
								vm.$message({
		    						message:"<%=rb.getString("ChengGong")%>",
		    						type:'success',
		    					})
                                enbFileVue.$refs.slide.hide();
                                enbFileVue.$refs.file_table.refresh();
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
			vm.init();
			eventBus.$off('edit-file').$on('edit-file',this.submit);
			eventBus.$off('cancel-edit').$on('cancel-edit',this.cancel);
		}
	}) 
</script>