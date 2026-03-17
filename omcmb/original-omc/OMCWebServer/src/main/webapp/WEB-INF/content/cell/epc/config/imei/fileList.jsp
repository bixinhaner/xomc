<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
.imeiFilePage {
	display: flex;
	width: 100%;
	height: 100%;
	overflow: hidden;
}
.importFileBoxCls {
	flex: 0 360px;
	height: 100%;
}
.imeiFile_main {
	flex: 1;
	overflow: auto;
}
.imeiFile_add {
	flex: 0 400px;
}


</style>

<%-- IMEI配置文件管理 --%>
<div id="imeiFilePage" class="imeiFilePage">
	<div class="imeiFile_main">
		<el-ctable ref="file_table"  :url="file_url" :height="height" :query-params="query_file_params" pagination="true" style='border: 1px solid #D5DCEC;border-top:none;height:calc(100% - 2px); border-radius: 0 0 8px 8px;'>
			<el-table-column label='' width="30" class-name="operationColumn">
				<template slot-scope="scope">
					<div class="el-icon el-icon-operation-more" @click="optClickFile(scope.row,event)" v-clickoutside="handerClose" ></div>
				</template>
			</el-table-column>
			
			<el-table-column label='<%=rb.getString("WenJianMing")%>' width="500" prop="fileName"></el-table-column>
			<el-table-column label='<%=rb.getString("ShangChuanZhe")%>' width="200" prop="uploader"></el-table-column>
			<el-table-column label='<%=rb.getString("ShangChuanShiJian")%>' width="200" prop="uploadTime"></el-table-column>
			<el-table-column label='<%=rb.getString("MiaoShu")%>' prop="description"></el-table-column>
			<template slot="toolbar">
			
				<div class='toolbarHeadBtnBoxCls' style="height:45px;position:relative;">
					<div v-if="isWritable" class="newIconBoxCls-bt" style="right:20px;top:5px;position:absolute;" @click="importFileClick" tip="<%=rb.getString("DaoRuWenJian")%>">
						<span class='el-icon el-icon-operation-import'></span>
					</div>
					<div class='queryGroup commonSearchWarp'>
						<el-input v-model='query_file_form.searchText' @keyup.enter.native="queryFile" class='pairgrid-query' placeholder='<%=rb.getString("WenJianMing")%>'></el-input>
						<i @click='queryFile' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
					</div>
				</div>
			</template>
		</el-ctable>

		<el-cmenu ref="menu_file" :data="menus_file" @click="clickMenuFile"></el-cmenu>
	</div>
	<div v-show="importFileShow" class="imeiFile_add">
		<div class="rightOutBoxHeadCls">
			<span>{{rightOutBoxTitle}}</span>
			<span class="el-icon el-icon-close greyIcon" @click="rightBoxClose"></span>
		</div>
		<div class="rightItemMainBox">
			<el-form ref="importFileForm" :model="importFileForm" label-position="top" :rules="importFileFormRules" :hide-required-asterisk=true>
				
				<el-form-item label="<%=rb.getString("WenJianMing")%>" prop="fileName">
					<span slot="label" class="labelSlotCls">
						<%=rb.getString("WenJianMing")%>
						<span v-show="importFileType == 'add'" >( <%=rb.getString("DangQianZhiChiWenJianLeiXing")%> )</span>
					</span>
					<el-upload 
						v-show="importFileType == 'add'" 
						:before-upload='beforeUpload'  
						:on-success='checkImportFile' 
						:on-change="importFileChange" 
						:show-file-list=false 
						ref="importFile" 
						:action="importFileForm.importFileUrl"
						:auto-upload="false"
						accept=".xlsx, .xls">
						<el-input :readonly="true" :value="importFileForm.fileName">
							<a slot="append" class="el-icon el-icon-operation-import greyIcon" @click="importFileSelect"></a>
						</el-input>
						<a slot="trigger" ref="file_up"></a>
					</el-upload>
					<el-input v-show="importFileType != 'add'" v-model='importFileForm.fileName' :disabled="true"></el-input>
				</el-form-item>
								
				<el-form-item label="<%=rb.getString("MiaoShu")%>" prop='desc'>
					<el-input v-model='importFileForm.desc' type='textarea' :rows='2' :disabled="importViewFlag"></el-input>
				</el-form-item>
			</el-form>
		</div>
		<div class="footer" v-show="importFileType !== 'view'" >
			<div class="lnkbuttonGroup"  style="margin-left:20px;" >
				<el-button type="primary" @click="downloadTemplate"><%=rb.getString("MuBanDaoChu")%></el-button>
				<el-button type="primary" @click="importFileSubmit"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="rightBoxClose"><%=rb.getString("QuXiao")%></el-button>
			</div>
		</div>
	</div>
</div>

<script type="text/javascript">
var imeiFilePage = new Vue({
	el:'#imeiFilePage',
	data(){
		var vm = this,
		
		fileNameValidate = (rule,value,callback) => {
			
			if(value == ""){
				callback(new Error("<%=rb.getString("QingXianXuanZeWenJian")%>"))
			}else if(!vm.fileFormatMatch(value,reg)){
				callback(new Error(message))
			}else{
				var pathSplit = value.split(/\\/),
					filename = pathSplit[pathSplit.length - 1];
				
				if(filename.length>100) {
					callback("<%=rb.getString("WenJianMingBuNengChaoGuoYaoQiu")%>");
				}else {
					callback();
				}
			}
		};
		return {
			height:"100%",
			/* file_url:[
				{
					"id":'1',
					"fileName":'file1',
					"uploader":"admin",
					"uploadTime":"2018-01-01 00:00:00",
					"description":"description1",

				}
			], */
			file_url:'${ctx}/cell/imei/queryImsiIMEIPageList.action',
			query_file_params:{
				timeZone:timeZone,
				searchText:'',
				rd: ''
			},
			query_file_form:{
				searchText:''
			},
			menus_file:[],
			rowDataFile:[],

			importFileShow:false,
			importFileType:'',
			importViewFlag:false,
			importFileForm:{
				
				file:'',
				fileName:'',
				desc:'',
				
			},
			importFileFormRules:{
				
			},
			
			fileErrorData:'',
			
			
		}
	},
    computed: {
        rightOutBoxTitle() {
			var vm = this,
				codes={
					'add':'File Import',
					'view':'File Information',
					'edit':'File Modify'
				};

			return codes[this.importFileType];
		},
        isWritable() {
        	return writableMap.CODE_ENB_DEVICE_HALOB == true;
        },
    },
	methods:{ 
		init:function(){
			
		},
		handerClose:function(){
			this.$refs.menu_file.hide();
		},
		queryFile(){
	    	Object.assign(this.query_file_params,this.query_file_form);
	    	this.query_file_params.rd = Math.random();
	    },
		optClickFile(row,ev){
			var vm = this;
			vm.rowDataFile = row;
			var TuiJian = '',
				
				modifyShow = false,
				deleteShow = true;

			vm.menus_file = [
				//{label:"<%=rb.getString("XinXi")%>",cls:"el-icon el-icon-operation-info",code:"view"},
				{label:"<%=rb.getString("XiaZai")%>",cls:"el-icon el-icon-operation-download",code:"download"},
				//{label:"<%=rb.getString("XiuGai")%>",cls:"el-icon el-icon-operation-edit CODE_ENB_DEVICE_HALOB hidden" ,code:"modify"},
				{label:"<%=rb.getString("ShanChu")%>",cls:"el-icon el-icon-operation-delete CODE_ENB_DEVICE_HALOB hidden" ,code:"del"},
				
			]
			vm.$nextTick(function(){
				document.body.click();
				vm.$refs.menu_file.show(ev);
			})
		},
		clickMenuFile(ev){
			var codes = {
					view:this.viewFile,
					modify:this.editFile,
					download:this.downloadFile,
					del:this.delFile,
					
			}
			if(codes[ev.code]){
				codes[ev.code](this.rowDataFile)
			}
		},
		viewFile(row){
			var vm = this;
			
			vm.importFileType = 'view';
			vm.importViewFlag = true;
			Object.keys(vm.importFileForm).forEach(function(key){
				if(key == 'fileName'){
					vm.importFileForm[key] = row.file_name ? row.file_name : '';
				}else{
					if(row[key] != undefined && row[key] != null){
						vm.importFileForm[key] = row[key];
					}
				}
			});
			vm.importFileShow = true;
		},
		editFile(row){
			var vm = this;
			
			vm.importFileType = 'edit';
			vm.importViewFlag = false;
			Object.keys(vm.importFileForm).forEach(function(key){
				if(key == 'fileName'){
					vm.importFileForm[key] = row.file_name ? row.file_name : '';
				}else{
					if(row[key] != undefined && row[key] != null){
						vm.importFileForm[key] = row[key];
					}
				}
			});
			vm.importFileShow = true;

		},
		downloadFile(){
			var vm = this;
			
			var params = {
				id : vm.rowDataFile.id,
					
			}
			
			var url = '${ctx}/cell/imei/downLoadImsiIMEIFile.action';
			exportByForm(url,params);
				
		},
		delFile(){
			var vm = this;
			var confirmStr = '<%=rb.getString("QueRenShanChu")%>';
			var url = '${ctx}/cell/imei/deleteImsiIMEIFile.action';
			var params = {
				fileIds :vm.rowDataFile.id
			}
			vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
				customClass:"warningConfirm",
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancelButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning',
				closeOnClickModal:false
			}).then(() => {
				axios.post(url,stringify(params)).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$message({
    						message:"<%=rb.getString("ChengGong")%>",
    						type:'success',
    					})
                        vm.$refs.file_table.refresh();
					}else{
						vm.$message.error(data["message"])
					}
				}).catch(() => {})
			})
		},
		
		importFileClick(){
	    	var vm = this;
			vm.importFileType = 'add';
			vm.importViewFlag = false;
			vm.$refs.importFileForm.resetFields();
			vm.importFileShow = true;
	    },
	   
		// 升级文件选择
		importFileChange(file, fileList) {
			var vm = this;
			vm.fileErrorData = '';
			vm.importFileForm.file = file.raw;
			vm.importFileForm.fileName = file.name;
			
		},
		/**
		* 文件上传之前
		* @param file{object}   文件信息
		*/ 
		beforeUpload(file){
			var vm = this;
			var fileName = file.name,fileSize = file.size;
			var fd = new FormData(),
				config = {
					headers: { 'Content-Type': 'multipart/form-data' },
					onUploadProgress:(ev)=>{
						if(ev.lengthComputable || ev.event.lengthComputable) {
							var total = ev.total,
								loaded = ev.loaded,
								percent = 100*loaded/total;
							$('#progressUploadFile').progressbar('setValue', percent.toFixed(2));
						}
					}
				};
		
			fd.append('uploadFile',file); //文件流
			fd.append('newFileName',fileName); //文件名

			fd.append('description',vm.importFileForm.desc);//描述

			fd.append('md5','');

			vm.fileErrorData = vm.$refs.importFile.uploadFiles[0];
			$('#progressUploadFile').progressbar('setValue', 0);// 将进度条进度置为0
			$("#winUploadPro").window("open");// 打开进度条窗口
			axios.post("${ctx}/cell/imei/importBatchIMEIInfos.action",fd,config).then(function(response){
				var data = response.data
				$("#winUploadPro").window("close");// 关闭进度条窗口
				if(data["success"]){
					vm.fileErrorData = '';
					$.messager.alert('<%=rb.getString("TiShi")%>','<%=rb.getString("ShangChuanChengGong")%>');
					vm.$refs.file_table.refresh();
					vm.rightBoxClose();
				}else{
					vm.$message.error(data["msg"]);
				}
			})
			
			return false;
		},
		//发送请求，校验device文件内容 
		checkImportFile(res, file) {    
			var vm = this;
			if (res.success) {
				if (res.suc_count > 0) {
					vm.$message({
						type: 'success',
						message: '<%=rb.getString("ChengGong")%>'
					});
				} else {
					vm.$message({
						type: 'warning',
						message: '<%=rb.getString("ShiBai")%>'
					});
				}
			} else {
				vm.$message({
					type: 'error',
					message: res.msg
				});
			}
			//修改已选择文件状态  
			var fileList = vm.$refs.importFile.uploadFiles;
			fileList.forEach(function (file) {
				file.status = 'ready';
			})
		},
		importFileSelect() { 
			var vm = this;
			vm.$refs.importFile.clearFiles();
			vm.$refs['file_up'].click();
		},
		// 文件校验
		fileFormatMatch(str,regs){
			var regsArr = regs.toLowerCase().split(",");
			if(str.substring(str.length-6) == 'tar.gz'){
				var suffix = "tar.gz";
			}else{
				var suffix = str.substring(str.lastIndexOf(".")+1).toLowerCase();
			}
			if(regsArr.indexOf(suffix)>-1){
				return true;
			}else{
				return false;
			}
		},
		downloadTemplate() {
			var url = '${ctx}/cell/imei/downloadIMEIConfigTemp.action',
				params = {
					timeZone: timeZone
				};

			exportByForm(url,params)
		},
		// 升级文件导入确定
		importFileSubmit(){
			var vm = this;
			vm.$refs.importFileForm.validate((valid) => {
				if(valid){
					if(vm.importFileType == 'add'){
						if(vm.fileErrorData){
							vm.$refs.importFile.uploadFiles.push(vm.fileErrorData);
						}
						vm.$refs.importFile.submit();
					}else{
						vm.fileImportEditSubmit()
					}
				}
			})
		},
		// 升级文件 修改提交
		fileImportEditSubmit(){
			var vm = this,
				
				params={
					versionId:vm.rowDataFile.id,
					fileName:vm.importFileForm.fileName,
					
					version:vm.importFileForm.version,
					
					desc:vm.importFileForm.desc,
					
				};
			
			axios.post("${ctx}/cell/version/goModifyDeviceVersionFileInfo.action",stringify(params)).then(function(response){
				var data = response.data;
				if(data) {
					if(data["success"]){
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type:'success'
						});
						vm.$refs.file_table.refresh();
						vm.rightBoxClose();
					}else{
						vm.$message.error(data["message"])
					}
				}
			}).catch(function(error){})
		},
		// 升级文件导入 关闭
		rightBoxClose(){
			var vm = this,
				params = {
					
					file:'',
					fileName:'',
					desc:'',
					
				};
			vm.importFileShow = false;
			Object.assign(vm.importFileForm,params)

		},
		fileTypeClick(event){
			var vm = this;
			
			event.preventDefault();
		},
		
	},
	mounted(){
		this.init();
		
	}
})
</script>