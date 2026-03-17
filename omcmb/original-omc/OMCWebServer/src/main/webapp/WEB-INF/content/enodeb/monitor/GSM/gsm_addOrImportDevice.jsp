<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style type="text/css">
	#gsmAddOrImport .tipText{
		margin-bottom: 20px;
		font-size: 12px;
		color: #C2C2C2;
	}
	.infoTip:before{
		color:#CFCFCF;
		font-size:14px;
		margin-right:6px;
	}
	.footerBoxCls{
		height:50px;
		display:flex;
		align-items:center;
		justify-content:flex-end;
		border-top:1px solid #E9EDF9;
		padding-right: 40px;
	}

	#importDeviceCard .el-card__body{
		padding:30px 30px 0;
		background:#FFF;
		border:none;
		height:186px;
	}
	#importDeviceCard .el-card__body .uploadBox .el-input__suffix{
		top:4px;
	}
	#importDeviceCard .el-card__body .el-upload__tip{
		margin-top:0;
	}
	#importDeviceCard .el-card__body .uploadFormat{
		margin-left:10px;
		font-size:12px;
		color:#999999;
	}
	#importDeviceCard .el-card__header{
		padding:0 20px;
		border-bottom:1px solid #E9E9E9;
		color:#333;
	}
	#importDeviceCard .el-card__footer{
		border:none;
		height:24px;
		line-height:24px;
		padding:0 30px 20px;
	}
	#importDeviceCard .el-card__footer{
		border-top:none;
	}
	.importResultWarp .importResultDivs {
	    margin-bottom: 16px;
	}
	.importResultWarp .downloadBtnTips {
	    display: block;
	    margin: 40px auto 0;
	}
</style>

<div id="gsmAddOrImport" style="position:relative;">
	<el-form :model='addOrImportForm' :rules="addOrImportRules" ref="addOrImportForm" label-position="top" style="padding:20px 40px 0px 40px;">
		<el-form-item  prop='type' label="" style='margin-bottom:20px;'>
			<span style="font-size:14px;color:#333333;margin-right:20px;"><%=rb.getString("TianJiaLeiXing")%></span>
			<el-radio-group v-model="addOrImportForm.type" size="small">
				<el-radio border label="enter" style='margin-right:30px;'><%=rb.getString("ShouDongShuRu")%></el-radio>
				<el-radio border label="import" style='margin-bottom:0px;'><%=rb.getString("PiLiangDaoRu")%></el-radio>
			</el-radio-group>
		</el-form-item>
		<div v-show="addOrImportForm.type == 'enter'">
			<el-form-item label='<%=rb.getString("Title_SheBeiBianMa")%>' prop='serialnumber' style="margin-bottom:15px;">
				<el-input type="textarea" style="width:70%" v-model="addOrImportForm.serialnumber"></el-input>
			</el-form-item>
			<div class="tipText">
				<span class="el-icon el-icon-circle-info infoTip"></span>
				<span><%=rb.getString("eNBZhuCeTiShiWenZi")%></span>
			</div>
		</div>
		<div v-show="addOrImportForm.type == 'import'" style="margin-bottom:20px;">
			<div style="display:flex;" class="uploadBox">
				<el-form-item label='<%=rb.getString("WenJian")%>' prop='' style="margin-bottom:15px;">
					<el-upload :on-success='checkFile' :on-change="fileChange" :show-file-list=false ref="upload"
							:action="uploadFileURL" :data="fileParams" name="uploadFile" :auto-upload="false">
						<el-input :readonly="true" :value=fileName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' style="width:260px;">
							<a slot="suffix" style="padding-top:4px;" class="el-icon el-icon-operation-import importBox" @click="fileSelect"></a>
						</el-input>
						<span class="uploadFormat">.xlsx /.csv</span>
						<div slot="tip" class="el-upload__tip" v-show="!typeFlag"><%=rb.getString("ZhiZhiChiXLSXCSVWenJian")%></div>
						<div slot="tip" class="el-upload__tip" v-show="selectFlag"><%=rb.getString("QingXianXuanZeWenJian")%></div>
						<a slot="trigger" ref="file_up"></a>
					</el-upload>	
				</el-form-item>
			</div>
			<div class="tipText">
				<span class="el-icon el-icon-circle-info infoTip"></span>
				<span><%=rb.getString("DaoRuWenJianTiShi")%></span>
			</div>
			<span style="cursor:pointer;" @click="exportTemplate">
				<span style='vertical-align:top' class='el-icon el-icon-common-download'></span>
				<span style='color:#363B4E;text-decoration:underline'><%=rb.getString("DaoChuMuBan")%></span>
			</span>
		</div>
		<el-form-item label='<%=rb.getString("SheBeiZuMingCheng")%>' prop='groupId'>
			<el-select v-model="addOrImportForm.groupId" >
				<el-option v-for="item in deviceGroupSelections" :key="item.id" :label="item.group_name" :value="item.id"></el-option>
			</el-select>
		</el-form-item>
					
	</el-form>
	<div slot='footer' class="footerBoxCls">
		<el-button-group size="mini">
			<el-button type="primary" size="mini" @click="addOrImportantSubmit"><%=rb.getString("QueDing")%></el-button>
			<el-button size="mini" onclick="document.body.click();"><%=rb.getString("QuXiao")%></el-button>
		</el-button-group>
	</div>	
	
	<el-dialog title='<%=rb.getString("JieGuo")%>' :visible.sync="importResultDialog" width="400" :close-on-click-modal="false" :modal-append-to-body="false" class='importResultWarp'>
        <p class='commonText14 importResultDivs'><%=rb.getString("SheBeiDaoRuJieGuo")%></p>
        <div class='importResultDivs commonFlex'>
            <p class='commonSize14'><%=rb.getString("DaoRuChengGongShuLiang")%></p>
            <p class='exportTemplateIcon' style='color: #67D972;'>{{checkSuccessCount}}</p>
        </div>
        <div class='importResultDivs commonFlex'>
            <p class='commonSize14'><%=rb.getString("DaoRuShiBaiShuLiang")%></p>
            <p class='exportTemplateIcon' style='color: #E88282;'>{{checkUnSuccessCount}}</p>
        </div>
        <el-button type="primary" @click='downloadFailClick' class='downloadBtnTips'><%=rb.getString("XiaZaiShiBaiMingXi")%></el-button>
    </el-dialog>
</div>

<script>
var gsmAddOrImportVue = new Vue({
	el: '#gsmAddOrImport',
	data(){
		var validatorNum = (rule,value,callback) => {
			var serialNumber = value,
				temp = /^(\d|[a-zA-Z]|-|\s){1,30}$/,
				list = serialNumber.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ 
					return item.length > 0;
				});
			
			if(this.addOrImportForm.type == 'enter'){
				if (serialNumber == null || serialNumber.length == 0) {
					callback(new Error('<%=rb.getString("SNBuNengWeiKong")%>'));
				}else {
					var nameFlag = list.every(function(item,index){
						return temp.test(item)
					})
					if(nameFlag){
						callback()
					}else{
						callback(new Error('<%=rb.getString("QingShuRuZhengQueSn")%>'));
					}
				}
			}else{
				callback();
			}
		};
		return {
			addOrImportForm:{
				type:'enter',
				serialnumber:'',
				groupId:''
			},
			addOrImportRules:{
				serialnumber:[
					{validator:validatorNum,trigger:'blur'}
				]
			},
			showAddDeviceCard:false,
			showImportCard:false,
			enodebForm:{
				serialnumber:'',
				groupId:''
			},
			enodebRules:{
				serialnumber:[
					{validator:validatorNum,trigger:'blur'}
				]
			},
			type:'',
			deviceGroupSelections:[],
			groupId:'',
			defaultGroupId:'',
			//导入文件
			importGroupId:'',
			
			selectFlag:false,        //标识是否选择了文件 
			typeFlag:true,           //校验已选择的文件格式 
			fileName:'',
			fileParams:{},            //上传文件时自定义的参数   
			uploadFileURL:'${ctx}/system/deviceGroup/uploadFile.action',

			importResultDialog: false,
			checkSuccessCount: '0',
			checkUnSuccessCount: '0'
		}
	},
	
	methods: {	
		addDeviceInit(){
				var vm = this;
				axios.post("${ctx}/system/deviceGroup/getDeviceGroupList.action").then((res) => {
					var data = res.data.rows;
					if(data){
						vm.deviceGroupSelections = data; 						
						vm.defaultGroupId = data[0].id;
						vm.addOrImportForm.groupId = vm.defaultGroupId;
					}					
				})				
			}, 
		
		closeAddDevice(){
			var vm = this;
			vm.openOrCloseAdd();
			vm.openOrCloseImport();
			document.body.click();
			
		},
		openOrCloseAdd(){
			var vm = this;
			vm.$refs.addOrImportForm.resetFields();
			vm.addOrImportForm.groupId = vm.defaultGroupId;
		},
		openOrCloseImport(){
			var vm = this;
			vm.typeFlag = true;
			vm.selectFlag = false;	
			vm.fileName = '';
			vm.fileParams.FileName = '';
			vm.fileParams.group_id = vm.defaultGroupId; 
		},
		
		/**
		* 文件上传成功函数 
		* @param res{object}   返回信息
		* @param file{object}  文件信息
		*/
		checkFile(res,file){    //发送请求，校验device文件内容 
			var vm = this;
			if(res.checkSuccess){
				//checkSuccess: true 时，设备注册提示走原有的逻辑根据 success 字段处理，成功即成功，失败即失败
				if(res.success){
					vm.$message({
						message: '<%=rb.getString("ChengGong")%>',
						type:'success',
					})
					vm.showImportCard = false;
					gsmvm.refreshList();//刷新列表
					vm.closeFileSelect();
					document.body.click();
				}else{
					vm.$message.error(res.msg)
				}
				vm.importResultDialog = false;
			}else{
				// false 查询导入结果; 当前导入含有失败情况时；
				vm.checkSuccessCount = res.checkSuccessCount;
				vm.checkUnSuccessCount = res.checkUnSuccessCount;
				vm.importResultDialog = true;
			}
			
			//修改已选择文件状态  
			var fileList = vm.$refs.upload.uploadFiles;
			fileList.forEach(function(file){
				file.status = 'ready';
			})
		},
		//导入文件结果-下载
        downloadFailClick(){
            var vm = this;

            vm.importResultDialog = false;
            exportByForm('${ctx}/system/deviceGroup/downloadFailureFile.action', { type: 'enb'})
        },
		/**
		* 选择文件后，校验格式，并赋值页面显示 
		* @param file{object}   文件信息
		* @param fileList{Array}  文件列表
		*/ 
		fileChange(file,fileList){ 
			var vm = this;
			vm.selectFlag = false;
			const typeFlag = file.name.substr(file.name.lastIndexOf("."))  === '.xlsx' || file.name.substr(file.name.lastIndexOf("."))  === '.csv'
			vm.typeFlag = typeFlag;
			var groudId = '';
			if(typeFlag){
				vm.fileName = file.name;					
			}else {
				vm.fileName = '';
			}
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
			vm.fileName = '';
			vm.typeFlag = true;
			vm.selectFlag = false;
			vm.$refs.upload.clearFiles();
		},
		// 导出设备模板
	    exportTemplate(){
	    	
	    	//cpe的设备模板导出
	    	var vm = this;
    	    var params = {};
    	    var url = "${ctx}/system/deviceGroup/downloadImportCellTemplate.action";

		    params.group_id = vm.defaultGroupId;//应按照默认设备组id 还是当前选中的设备组id
	    	params.search_text = ''; //此参数是否有效
    	    params.like_fields = "serial_number";
        	var bool = checkParams(params)
			if(!bool) return false;
        	exportByForm(url,params)
	    },
		// 导入设备文件确定
		uploadDevice(){
			var vm = this;
			vm.fileParams.FileName = vm.fileName;
			vm.fileParams.group_id = vm.importGroupId;
			if(vm.fileParams.FileName){
				vm.$refs.upload.submit();
			}else{
				vm.typeFlag = true;
				vm.selectFlag = true;
			}			
		},
		addOrImportantSubmit(){
			var vm = this,
				snStr = vm.addOrImportForm.serialnumber || '',
				list = snStr.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ return item.length > 0;});
			
			if(vm.addOrImportForm.type == 'enter'){
				vm.$refs["addOrImportForm"].validate((valid) => {
					if(valid){
						axios.post("${ctx}/system/deviceGroup/addAndAssignEnb.action",stringify({
							"group_id":vm.addOrImportForm.groupId,
							"serialNumber": list.join(";"),
							"device_status": '1'
						})).then(function(response){
							let data = response.data;
							if (data.success){
								gsmvm.refreshList();
								vm.closeAddDevice();
								vm.$message({
									message: '<%=rb.getString("TianJiaSheBeiChengGong")%>',
									type:'success',
								})
							}else {
								vm.$message.error(data.message)
							}
						}).catch(function(error){})
					}else{
					}
				})
			}else{
				vm.fileParams.FileName = vm.fileName;
				vm.fileParams.group_id = vm.addOrImportForm.groupId;
				if(vm.fileParams.FileName){
					vm.$refs.upload.submit();
				}else{
					vm.typeFlag = true;
					vm.selectFlag = true;
				}
			}
				
		}
	},
	mounted() {
		this.addDeviceInit();
	}
});

</script>