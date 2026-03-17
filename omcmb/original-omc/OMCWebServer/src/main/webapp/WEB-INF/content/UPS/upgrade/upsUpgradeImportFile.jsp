<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
#upsUpgradeImportfile .el-form-item__label{
	line-height:26px;
	width:140px;
	text-align:left;
}
#upsUpgradeImportfile .el-form-item__error{
	margin-left:140px;
}
#upsUpgradeImportfile .el-form-item{
	margin-bottom:35px;
}
#upsUpgradeImportfile .el-select .el-input.is-disabled .el-input__inner{
	min-height:26px;
	max-height:26px;
}
#upsUpgradeImportfile .uploadInput .el-input__suffix{
	top:5px !important;
}
</style>

<div id='upsUpgradeImportfile'>
	<el-form ref="upsFileImportForm" :model="importForm" :rules="importRules" style="margin-left:30px;margin-top:30px;" :hide-required-asterisk=true>
		<el-form-item prop="product" label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" style='display:inline-block;width:380px;'>
			<el-input v-model='importForm.product' disabled=true></el-input>
		</el-form-item>
		<el-form-item label="<%=rb.getString("WenJianMing")%>" prop="fileName">
			<el-upload :before-upload='beforeUpload' :on-success='checkFile' :on-change="fileChange"  :show-file-list=false ref="upload"
					:action="uploadFileUrl" :data="fileParams" name="uploadFile" :auto-upload="false">
				<el-input style="width:400px;" class="uploadInput" :readonly="true" :value='importForm.fileName' placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>'>
					<a slot="suffix" class="el-icon el-icon-operation-import importBox" @click="fileSelect"></a>
				</el-input>								
				<a slot="trigger" ref="file_up"></a>
			</el-upload>
		</el-form-item>
		<el-form-item label="<%=rb.getString("BanBen")%>" prop="version">
			<el-input v-model='importForm.version' :disabled="viewFlag" style="width:400px;"></el-input>
		</el-form-item>
		
		<el-form-item label='<%=rb.getString("TuiJian")%>' prop='recommend'>
			<el-select v-model='importForm.recommend' :disabled="viewFlag">
				<el-option label='<%=rb.getString("Shi")%>' value='1'></el-option>
				<el-option label='<%=rb.getString("Fou")%>' value='0'></el-option>
			</el-select>
		</el-form-item>
		<el-form-item label="<%=rb.getString("MiaoShu")%>" prop='desc'>
			<el-input :disabled="viewFlag" v-model='importForm.desc' type='textarea' :rows='6' style='width:600px;'></el-input>
		</el-form-item>
	</el-form>
</div>
<script>
	new Vue({
		el:'#upsUpgradeImportfile',
		data(){
			var versionValidate = function(rule,value,callback) {
				if(value) {
					if(value.length>45) {
						callback('<%=rb.getString("FanWei")%>:1-45 <%=rb.getString("ZiFuFuShu")%>');
					}else {
						callback();
					}
				}else {
					callback('<%=rb.getString("QingShuRuWenJianBanBen")%>');
				}
			};
			var fileNameValidate = function(rule,value,callback) {
				if(value) {
					if(value.length>100) {
						callback('<%=rb.getString("WenJianMingBuNengChaoGuoYaoQiu")%>');
					}else {
						callback();
					}
				}else {
					callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
				}
			};
			return{
				importForm:{
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
					fileName:[
						{validator: fileNameValidate}
					],
					product:[
						{required:true,message:'<%=rb.getString("ShuRuBiTianXiang")%>',trigger:'change'}
					],
					recommend:[
						{required:true,message:'<%=rb.getString("ShuRuBiTianXiang")%>',trigger:'change'}
					]
				},
				viewFlag : false,
				fileParams:{},            //上传文件时自定义的参数  
				fileList:[],
				uploadFileUrl:'',
				selectFlag:false,
			}
		},
		methods:{
			/**
			* 文件上传成功函数 
			* @param res{object}   返回信息
			* @param file{object}  文件信息
			*/
			checkFile(res,file){    //发送请求，校验device文件内容 
				
			},
			/**
			* 选择文件后，校验格式，并赋值页面显示 
			* @param file{object}   文件信息
			* @param fileList{Array}  文件列表
			*/ 
			fileChange(file,fileList){ 
				var vm = this;
				vm.selectFlag = false;
				vm.importForm.fileName = file.name;
				vm.importForm.version = file.name.substring(0,file.name.lastIndexOf("."));
				vm.fileParams.FileName = file.name;
			},
			
			// 选择文件
			fileSelect(){  
				var vm =this;
				vm.$refs.upload.clearFiles();
				vm.$refs['file_up'].click();
			},
			
			// 移除导入文件
			closeFileSelect(){
				var vm = this;
				vm.importForm.fileName = '';
				vm.selectFlag = false;				
				vm.$refs.upload.clearFiles();
			},
						
			/**
			* 文件上传之前
			* @param file{object}   文件信息
			*/ 
			beforeUpload(file){
				var vm = this;
				var fileName = file.name,fileSize = file.size,fileType = 'ups';
				var fd = new FormData(),
					config = {
						headers: { 'Content-Type': 'multipart/form-data' }
					};
				fd.append('uploadFile',file); //文件流
				fd.append('newFileName',fileName);//文件名
				fd.append('fileSize',fileSize);//文件大小
				fd.append('fileType',fileType); // 文件类型
				fd.append('desc',vm.importForm.desc);//描述
				fd.append('product',vm.importForm.product);
				fd.append('version',vm.importForm.version);
				fd.append('recommend',vm.importForm.recommend);
                fd.append('to_who','all');
				
				axios.post("${ctx}/cell/version/uploadVersionFile.action",fd,config).then(function(response){

					var data = response.data
					if(data["MD5"]){
						
						$.messager.alert(TiShi,"<%=rb.getString("ShangChuanChengGong")%><%=rb.getString("DouHao")%><%=rb.getString("WenJianMD5Zhi")%><%=rb.getString("MaoHao")%>"+data["MD5"]);
						eventBus.$emit('hide-upsUpgrade-slide');
					}else{
						vm.$message.error(data["message"])
					}
				})
				
				return false;
			},
			
			/*确定导入*/
	        uploadDevice() {
				var vm = this,
				fileName;
				if(vm.importForm.fileName === ''){
					fileName = false
                }else{
                	fileName = true
                }

				vm.$refs.importForm.validate((valid) => {
                    if (valid && fileName) {
                    	vm.$refs.upload.submit();
                    	
                    } else {
                    	vm.selectFlag = true;
                    }
                }) 				
			},
			cancel(){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>'
				if(isFormChanged(vm.$refs.upsFileImportForm)){
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
				if(vm.importForm.fileName === ''){
					vm.selectFlag = true;
                }else{
                	vm.selectFlag = false;
                }
				vm.$refs.upsFileImportForm.validate((valid) => {
					if(valid && !vm.selectFlag){
						vm.$refs.upload.submit();
					}
				})
			}
		},
		mounted(){
			var vm = this;
			eventBus.$off('ups-upgrade-importSubmit').$on('ups-upgrade-importSubmit',this.submit);
			eventBus.$off('ups-upgrade-cancelImport').$on('ups-upgrade-cancelImport',this.cancel);
		}
	}) 
</script>