<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#gsmConfigurationPage{
		background-color: #FFFFFF;
		height: 100%;
		width: 100%;
		overflow: auto;
		position: relative;
	}
	#gsmConfigurationPage .loading::before{
		background-color: rgba(255,255,255,1);
	}
	#gsmConfigurationPage .deviceRecycleBinHeaderCls{
		position:relative;
	}
	#gsmConfigurationPage .el-ctable-toolbar{
		padding: 0px!important;
	}
	#gsmConfigurationPage .deviceListTableCls{
		width: 100%;
		height: 100%;
		overflow: hidden;
		position:relative;
	}
    .gsmConfigurationImportDialogCls .el-input__suffix{
        display: flex;
        align-items: center;
    }
    .gsmConfigurationImportDialogCls .el-form-item__content{
        line-height: 36px;
    }
    .gsmConfigurationImportDialogCls .el-ctable-toolbar{
        padding: 0px!important;
    }
    .gsmConfigurationFileInfoDialogCls .el-dialog__body{
        position: relative;
    }
    #gsmConfigurationPage .statusIconBoxCls{
        display: flex;
        align-items: center;
    }
    #gsmConfigurationPage .blueIcon::before{
		font-size: 16px!important;
		color: #4D84FF;
	}
	#gsmConfigurationPage .redIcon::before{
		font-size: 16px!important;
		color: #E88282;
	}
	#gsmConfigurationPage .greenIcon::before{
		color: #67D972;
		font-size: 16px!important;
	}
    #gsmConfigurationPage .blackIcon::before{
		color: #000000;
		font-size: 16px!important;
	}
</style>
<div id='gsmConfigurationPage'>
	<div class="container" style="min-width: 900px;">
		<div class="deviceListTableCls">
            <!-- 按钮   批量导入 -->
            <div class="newIconBoxCls-bt last-bt" style="right:20px;top: 5px;" @click="batchImportFile" tip="<%=rb.getString("PiLiangDaoRu")%>">		
                <span class="el-icon el-icon-operation-import"></span>
            </div>
			<el-ctable
				id="deviceTable"
				ref="ctableDevice"
				:url="deviceUrl"
				:height="height"
				:row-key="'smallCellCode'"
				:query-params="queryParams"
                :time="6"
				pagination="true"
				:rownumber=true
				:limit="limitBatch"
				@selection-change='batchSelect'
				>
                <template slot="toolbar">
                    <div class="deviceRecycleBinHeaderCls">
                        <div class="toolbarHeadBtnBoxCls">
                            <div class="selectBlukBoxCls">
                                <div class="selectMain">
                                    <div class="bulkSelectBtnBoxCls"  @click="openBulkSelectTable">
                                        <span class="el-icon-selected el-icon"></span>
                                        <span class="bulkSelectNumBoxCls">( {{deviceSelection.length}} )</span>
                                    </div>
                                    <div class="selectTableBoxCls" style="position: absolute;top: 38px;left: 0px;" v-show="bulkSelectShow">
                                        <div class="selectBoxTitle">
                                            <span><%=rb.getString("YiXuan")%></span>
                                            <span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectTable"></span>
                                        </div>
                                        <div class="selectBoxMain">
                                            <div class="tableInfoCls">
                                                <div class="tableInfoHeader">
                                                    <div><%=rb.getString("Title_SheBeiBianMa")%></div>
                                                    <div @click="clearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span>Clear</div>
                                                </div>
                                                <el-ctable
                                                    id="bulkSelectTable"
                                                    ref="bulkSelectTable"
                                                    :data="deviceSelection"
                                                    :showHeader="false"
                                                    :rownumber="false"
                                                    :front-pagination="true"
                                                    :row-key="'smallCellCode'"
                                                    height="270px" pagination="true" >
                                                    <el-table-column prop="alarm_id" v-if="false"></el-table-column>
                                                    <el-table-column width="588">
                                                        <template slot-scope="scope" >
                                                            <div class="tableItemCls">
                                                                <span>{{scope.row.serialNumber}}</span>
                                                                <span @click="delBulkSelected(scope.row)" class="el-icon el-icon-circle-close item_show"></span>
                                                            </div>
                                                        </template>
                                                    </el-table-column>
                                                </el-ctable>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </div>
                            <div :class="deviceSelection.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="backupFormEnbFile">
                                <span class="el-icon el-icon-circle-backup"></span>
                                <span><%=rb.getString("CongJiZhanBeiFen")%></span>
                            </div>
                            <div :class="deviceSelection.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="sendToEnbFile">
                                <span class="el-icon el-icon-operation-issued"></span>
                                <span><%=rb.getString("XiaFaDaoJiZhanPeiZhi")%></span>
                            </div>
                            <div :class="deviceSelection.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="batchDownload">
                                <span class="el-icon el-icon-operation-download"></span>
                                <span><%=rb.getString("PiLiangXiaZai")%></span>
                            </div>
                        </div>
                        <div class="tableHeadBoxCls" style="position:relative;">
                            <div id="tableHeadQuery" class="tableHeadQueryBoxCls">
                                <div class="headQueryBox">
                                    <div class="queryGroup">
                                        <el-input v-model="searchText" @keyup.enter.native="query" @focus="queryInputFocus" @blur="queryInputBlur" :placeholder='placeholderText' style="width:260px;"></el-input>
                                        <i @click='query' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
                                    </div>
                                </div>
                                <el-popfilter style="margin: 0 5px;"
                                    label='<%=rb.getString("ZhuangTai")%>'
                                    v-model="queryParams.status"
                                    type="single"
                                    :list="statusOptions">
                                </el-popfilter>
                            </div>
                        </div>
                    </div>
                </template>
				<!--设备列表-->
				<el-table-column width="50" type="selection" prop="ck" :reserve-selection="true"></el-table-column>
                <el-table-column label='' width="40" prop="">
                    <template slot-scope="scope">
                        <div class="el-icon el-icon-operation-info grayIcon" @click="fileDetailInfo(scope.row,event)" style="cursor: pointer;"></div>
                    </template>
                </el-table-column>
                <el-table-column prop="connection_status" width="45">
                    <template slot-scope="scope">
                        <div :class="{
                            'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
                            '':scope.row.have_connected==2,
                            'conn_exc':scope.row.connection_status=='Exception',
                            'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
                    </template>
                </el-table-column>
				<el-table-column label='<%=rb.getString("SheBeiWeiYiBiaoZhi")%>' min-width="120" prop="serialNumber"></el-table-column>
				<el-table-column label='<%=rb.getString("BSCMingCheng")%>' min-width="120" prop="hostName"></el-table-column>
				<el-table-column label='<%=rb.getString("ChanPinLeiXingBiaoZhi")%>' min-width="120" prop="productName"></el-table-column>
				<el-table-column label='<%=rb.getString("WenJianMing")%>' min-width="100" prop="fileName"></el-table-column>
				<el-table-column label='<%=rb.getString("ZhuangTai")%>' min-width="120" prop="status">
                    <template slot-scope="scope">
                        <div v-if="scope.row.status == '0'" class="statusIconBoxCls">
							<span class="el-icon el-icon-status-failed redIcon" style='margin-right:5px;'></span><span><%=rb.getString("XiaFaShiBaiZhuangTai")%>,{{scope.row.failureReason}}</span>
						</div>
						<div v-if="scope.row.status == '1'" class="statusIconBoxCls">
							<span class="el-icon el-icon-status-success greenIcon"></span><span style='margin-left:5px;'><%=rb.getString("XiaFaChengGongZhuangTai")%></span>
						</div>
                        <div v-if="scope.row.status == '2'" class="statusIconBoxCls">
							<span class="el-icon el-icon-status-inProgress blueIcon"></span><span style='margin-left:5px;'><%=rb.getString("XiaFaZhongZhuangTai")%></span>
						</div>
                        <div v-if="scope.row.status == '3'" class="statusIconBoxCls">
							<span class="el-icon el-icon-status-notstarted blackIcon"></span><span style='margin-left:5px;'><%=rb.getString("WeiXiaFaZhuangTai")%></span>
						</div>
                        <div v-if="scope.row.status == '4'" class="statusIconBoxCls">
							<span class="el-icon el-icon-status-failed redIcon" style='margin-right:5px;'></span><span><%=rb.getString("BeiFenShiBaiZhuangTai")%>,{{scope.row.failureReason}}</span>
						</div>
                        <div v-if="scope.row.status == '5'" class="statusIconBoxCls">
							<span class="el-icon el-icon-status-success greenIcon"></span><span style='margin-left:5px;'><%=rb.getString("BeiFenChengGongZhuangTai")%></span>
						</div>
                        <div v-if="scope.row.status == '6'" class="statusIconBoxCls">
							<span class="el-icon el-icon-status-inProgress blueIcon"></span><span style='margin-left:5px;'><%=rb.getString("BeiFenZhongZhuangTai")%></span>
						</div>
                    </template>
                </el-table-column>
				<el-table-column label='<%=rb.getString("ShiJian")%>' min-width="100" prop="taskTime"></el-table-column>
			</el-ctable>
		</div>
	</div>
    <!-- 导入文件框 -->
	<el-dialog title='<%=rb.getString("PiLiangDaoRu")%>' :visible.sync="showConfirmInfo" width="1100" class="gsmConfigurationImportDialogCls"
		append-to-body :close-on-click-modal="false" @close="closeImport">
		<el-form :model="importForm" ref="importForm" label-position="left" :rules='importRules'> 
            <div style="color:#FF4614;font-size:12px;margin-bottom: 10px;"><%=rb.getString("PiLiangDaoRuPeiZhiTiShi")%></div>
			<div style="padding-top:10px;border:1px solid #E9E9E9;" class="selectDeviceTable">
                <el-ctable
                    id="importDeviceTable"
                    ref="importCtableDevice"
                    :url="importDeviceUrl"
                    height="246px"
                    :row-key="'smallCellCode'"
                    :query-params="importQueryParams"
                    :time="6"
                    pagination="true"
                    :rownumber=true
                    :limit="'200'"
                    @selection-change="batchImportDevicesChange" 
                >
                     <!-- 高级查询 -->
				    <template slot="toolbar">
                        <div class='commonQuery' style='display: flex;align-items: center;height:45px;'>
                            <el-query type="normal" @query="queryImportDeviceTable" placeholder="<%=rb.getString("SheBeiWeiYiBiaoZhi")%>"></el-query>  
                            <el-popfilter style="margin: 0 5px;"
                                label='<%=rb.getString("ChanPinLeiXingBiaoZhi")%>'
                                v-model="importQueryParams.product"
                                type="single"
                                :list="productOptions">
                            </el-popfilter>
                        </div>
                    </template>
                    <!--设备列表-->
                    <el-table-column width="50" type="selection" prop="ck"></el-table-column>
                    <el-table-column label='<%=rb.getString("SheBeiWeiYiBiaoZhi")%>' min-width="120" prop="serialNumber"></el-table-column>
                    <el-table-column label='<%=rb.getString("BSCMingCheng")%>' min-width="120" prop="hostName"></el-table-column>
                    <el-table-column label='<%=rb.getString("ChanPinLeiXingBiaoZhi")%>' min-width="120" prop="productName"></el-table-column>
			    </el-ctable>
			</div>
            <el-form-item prop="devices"></el-form-item>
            <el-form-item label='<%=rb.getString("DaoRuWenJian")%>' style="margin: 10px 0px 0px 0px;" label-width="130px" prop="fileName" class='importSelect'>
	            <el-upload 
	            	ref="importFile"
	            	:before-upload='beforeUpload' 
	            	:on-success='checkFile' 
	            	:on-change="fileChange"  
	            	:show-file-list=false 	            	
				    :action="importForm.uploadFileUrl" 
				    :data="fileParams" 
				    name="uploadFile" 
				    :auto-upload="false" 
				    accept=".xml">
					<el-input :readonly="true" :value=fileName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' style="width: 270px;">
						<a slot="append" class="el-icon el-icon-operation-import importBox" @click="fileSelect"></a>						
					</el-input>	
					<span class="fileImportTypeTip"><%=rb.getString("DangQianZhiChixmlGeShiAll")%></span>								
					<a slot="trigger" ref="file_up"></a>
				</el-upload>	
            </el-form-item>
            
		</el-form>
		
		<div slot="footer" class="dialog-footer">
			<div class="buttonGroup">
				<el-button type="primary" @click="importSubmit"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="closeImport"><%=rb.getString("QuXiao")%></el-button>
			</div>	
		</div> 	
	</el-dialog>
    <!-- 文件详情弹窗 -->
    <el-dialog :visible.sync="fileDetailInfoShow" append-to-body top="10vh" class="gsmConfigurationFileInfoDialogCls">
        <div slot="title">
            <span class="el-dialog__title"><%=rb.getString("XinXi")%></span>
        </div>
        <div :class="fileDetailLoading ? 'loading' : ''">
            <el-input v-model="fileDetailContent" type="textarea" rows="25" readonly class="none-border"></el-input>
        </div>
    </el-dialog>
</div>
<script type="text/javascript">
var gsmConfigurationVue = new Vue({
	el:'#gsmConfigurationPage',
	data(){
        var vm = this;
		var validateDevice = function(rule,value,callback) { // 校验设备
                if(value.length == 0) {
                    callback('<%=rb.getString("QingXuanZeSheBei")%>');
                }else {
                    callback();
                }
			},
            validateFileName = function(rule,value,callback) { // 校验设备
                value = vm.fileName;
                if( value === '' || value === null || value === undefined) {
                    callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
                }else if(!fileFormatMatch(value,"xml")){
                    callback(new Error("<%=rb.getString("DangQianZhiChixmlGeShiAll")%>"))
                }else {
                    callback();
                }
            };
        return {
			bulkSelectShow:false,
			searchText:'',
			placeholderText:'<%=rb.getString("QingShuRu")%>',
			deviceUrl:'',
			height:'100%',
			queryParams:{
				searchText:'',
                product:'',
				timeZone:timeZone,
                like_fields:'serial_number'
			},
			batchOperation:batchOperation,
			deviceSelection:[],
			statusOptions:[
                {label:'<%=rb.getString("QuanBu")%>',value:''},
                {label:'<%=rb.getString("XiaFaShiBaiZhuangTai")%>',value:'0'},
                {label:'<%=rb.getString("XiaFaChengGongZhuangTai")%>',value:'1'},
                {label:'<%=rb.getString("XiaFaZhongZhuangTai")%>',value:'2'},
                {label:'<%=rb.getString("WeiXiaFaZhuangTai")%>',value:'3'},
                {label:'<%=rb.getString("BeiFenShiBaiZhuangTai")%>',value:'4'},
                {label:'<%=rb.getString("BeiFenChengGongZhuangTai")%>',value:'5'},
                {label:'<%=rb.getString("BeiFenZhongZhuangTai")%>',value:'6'},
            ],
            productOptions:[],

            showConfirmInfo:false,
            importDeviceUrl:'',
            importQueryParams:{
                searchText:'',
                product:'',
                timeZone:timeZone,
            },
			fileParams:{},              
            fileName:'',	            					
			importForm: {
                uploadFileUrl: '',
                devices: '',
           	},
           	importRules: {           		
           		fileName:[
                	{required:true, validator: validateFileName},
                ],
                devices:[
                    {validator: validateDevice}
                ],        
            },
            fileErrorData:'',
            fileDetailInfoShow:false,
            fileDetailContent:'',
            fileDetailLoading:false
		}
	},
    computed:{
		limitBatch(){
			return this.batchOperation ? '' : 1;
		},
	},
	methods:{
		// 初始化
		init(){
			var vm = this;
			vm.deviceUrl = '${ctx}/task/BatchConfigurationFile/queryBatchConfigFileTaskPageList.action';
			vm.queryProductOption();
		},
		// 查询设备组
		queryProductOption(){
			var vm = this;
			axios.post('${ctx}/task/BatchConfigurationFile/getSelectProductList.action').then(function(response){
				let data = response.data
				if(data && data.length>0){
					var productOptions = [{label:'<%=rb.getString("QuanBu")%>',value:''}];
					data.map((item,index)=>{
						var obj = {};
						obj.label = item.text;
						obj.value = item.value;
						productOptions.push(obj);
					})
				}
				vm.productOptions = productOptions;
			}).catch(function(error){});
		},
		/**
		* 选择的批量数据
		* @param selection:传入批量数据对象
		*/
	    batchSelect(selection){
	    	var vm = this;
	    	vm.deviceSelection = selection;
	 	},
		// 打开已选弹窗
		openBulkSelectTable(){
			var vm = this;
			vm.bulkSelectShow = true
		},
		// 关闭已选弹窗
		closeBulkSelectTable(){
			var vm = this;
			vm.bulkSelectShow = false;
		},
		// 设备已选表格 清空事件
		clearBulkSelected(){
			var vm = this;

			vm.$refs.ctableDevice.clearSelection();
		},
		// 设备已选表格 单个删除事件
		delBulkSelected(rows){
			var vm = this,
				tabs = 'ctableDevice',
				rowKey = 'smallCellCode';

			vm.deviceSelection = vm.deviceSelection.filter((items)=>{
				return items[rowKey] != rows[rowKey]
			});
			var selection = this.$refs[tabs].$refs.ctableInner.store.states.selection,
				irow= selection.filter((items)=>{
					return items[rowKey] == rows[rowKey]
				})[0];
			vm.$refs[tabs].toggleRowSelection(irow,false);
			var idx = vm.$refs[tabs].ckList.indexOf(rows[rowKey]);
			vm.$refs[tabs].ckList.splice(idx,1);
		},
        // 模糊搜索
		query(){
			var vm = this;
			vm.queryParams.searchText = this.searchText;
		},
        // 导入弹窗设备列表查询
        queryImportDeviceTable(val){
            var vm = this;
            vm.importQueryParams.searchText = val;
        },
		// 搜索域聚焦事件
		queryInputFocus(){
			var vm = this;
			vm.placeholderText = '<%=rb.getString("Title_SheBeiBianMa")%>';
		},
		// 搜索域失焦事件
		queryInputBlur(){
			var vm = this;
			vm.placeholderText = '<%=rb.getString("QingShuRu")%>';
		},
		// 从设备侧同步配置文件到OMC
		backupFormEnbFile(){
			var vm = this,
	    		params = {},
	    		url = '${ctx}/task/BatchConfigurationFile/addBatchBackupConfigFileTask.action',
				idsList = [];
			if(vm.deviceSelection.length <= 0)return
			vm.deviceSelection.map((item,index) => {
				idsList.push(item.smallCellCode)
			})
    	    params.smallCellCodes = idsList.join(',');
			axios.post(url,stringify(params)).then(function(response){
                let data = response.data;
                if ( data.success ){
                    vm.$message({
                        message: '<%=rb.getString("ChengGong")%>' ,
                        type:'success',
                    })
                    vm.$refs.ctableDevice.refresh();
                    vm.$refs.ctableDevice.clearSelection();
                }else {
                    vm.$message.error(data.message)
                }
            }).catch(function(error){})
		},
		// 下发配置文件到设备
		sendToEnbFile(){
			var vm = this,
	    		params = {},
	    		urls = '${ctx}/task/BatchConfigurationFile/addBatchSendConfigFileTask.action',
				idsList = [];
			if(vm.deviceSelection.length <= 0)return
			vm.deviceSelection.map((item,index) => {
				idsList.push(item.smallCellCode)
			})
    	    params.smallCellCodes = idsList.join(',');
            vm.$confirm('<%=rb.getString("QueRenXiaFa")%>','<%=rb.getString("QueRen")%>',{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(() => {
                axios.post(urls,stringify(params)).then(function(response){
                    let data = response.data;
                    if ( data.success ){
                        vm.$message({
                            message: '<%=rb.getString("ChengGong")%>' ,
                            type:'success',
                        })
                        vm.$refs.ctableDevice.refresh();
                        vm.$refs.ctableDevice.clearSelection();
                    }else {
                        vm.$message.error(data.message)
                    }
                }).catch(function(error){})
            }).catch(() => {})	
	    	
		},
        // 批量下载配置文件
        batchDownload(){
            var vm = this,
	    		params = {},
	    		urls = '${ctx}/task/BatchConfigurationFile/downLoadConfigFile.action',
				idsList = []
			if(vm.deviceSelection.length <= 0)return
			vm.deviceSelection.map((item,index) => {
				idsList.push(item.smallCellCode)
			})
    	    params.smallCellCodes = idsList.join(',');
            exportByForm(urls, params);
            vm.$refs.ctableDevice.clearSelection();
        },
        // 批量导入配置文件
		batchImportFile(){
			var vm = this;
			vm.showConfirmInfo = true;
            vm.importDeviceUrl = '${ctx}/task/BatchConfigurationFile/queryBatchConfigFileTaskPageList.action';
		},
        /**
         * 批量导入配置文件设备选择变化时，更新选择设备记录
         * @param value:
        */
        batchImportDevicesChange(selection) {
            var vm = this;
            vm.batchImportDevicesSelection = selection ? selection : [];
            vm.importForm.devices = vm.batchImportDevicesSelection.map(function(row){ return row.smallCellCode ;}).sort().join(',');
        },
        /**
        * 文件上传成功函数 
        * @param res{object}   返回信息
        * @param file{object}  文件信息
        */
        checkFile(res,file){    //发送请求，校验device文件内容 
            var vm = this;
            if(res.success){
                if(res.suc_count>0){
                    vm.$message({
                        type: 'success',
                        message: '<%=rb.getString("ChengGong")%>'
                    });
                }else {
                    vm.$message({
                        type: 'warning',
                        message: '<%=rb.getString("ShiBai")%>'
                    });
                }
                vm.showConfirmInfo = false;
                vm.$refs.ctableDevice.refresh();
                vm.closeFileSelect();
            }else{
                vm.$message({
                    type: 'error',
                    message: res.msg
                });
            }
            //修改已选择文件状态  
            var fileList = vm.$refs.importFile.uploadFiles;
            fileList.forEach(function(file){
                file.status = 'ready';
            })
        },
        /**
        * 选择文件后，校验格式，并赋值页面显示 
        * @param file{object}   文件信息
        * @param fileList{Array}  文件列表
        */ 
        fileChange(file,fileList){ 
            var vm = this;
            vm.fileName = file.name;
            vm.fileParams.fileName = file.name;
            vm.fileErrorData = '';
        },
        // 选择文件
        fileSelect(){  
            var vm =this;
            vm.fileName = '';
            vm.$refs.importFile.clearFiles();
            vm.$refs['file_up'].click();
        },
        // 移除导入文件
        closeFileSelect(){
            var vm = this;
            vm.fileName = '';			
            vm.$refs.importFile.clearFiles();
        },
        /**
        * 文件上传之前
        * @param file{object}   文件信息
        */ 
        beforeUpload(file){
            var vm = this,
                fileName = file.name,
                fileSize = file.size,
                fd = new FormData(),
                config = {
                    headers: { 'Content-Type': 'multipart/form-data' }
                },
                codesList = [];
            var cellCodes = '';
           
			if(vm.batchImportDevicesSelection.length <= 0)return
			vm.batchImportDevicesSelection.map((item,index) => {
				codesList.push(item.smallCellCode)
			})
            cellCodes = codesList.join(',');
            vm.fileErrorData = vm.$refs.importFile.uploadFiles[0];
            fd.append('uploadConfigFile',file); //文件流
            fd.append('fileName',fileName);//文件名
            fd.append('fileSize',fileSize);//文件名	大小
            fd.append('smallCellCodes',cellCodes);//基站编码
            console.log(file)
            axios.post("${ctx}/task/BatchConfigurationFile/uploadConfigFile.action",fd,config).then(function(res){
                if(res.data["success"]){	
                    vm.$message.success('<%=rb.getString("ChengGong")%>');
                    vm.$refs.ctableDevice.refresh();						
                    vm.showConfirmInfo = false;
                    vm.closeImport();
                    vm.fileErrorData = '';
                }else{
                    vm.$message.error(res.data["message"]);
                    vm.showConfirmInfo = false;
                }
            })
            return false;
        },
        /*确定导入*/
        importSubmit() {
            var vm = this;
            
           vm.$refs.importForm.validate((valid) => {
                if (valid) {
                    if(vm.fileErrorData){
                        vm.$refs.importFile.uploadFiles.push(vm.fileErrorData);
                    }
                    vm.$refs.importFile.submit();                  	
                }
            }) 	
        },
        // 关闭导入弹出框
        closeImport(){
            var vm = this;
            vm.showConfirmInfo = false;	
            vm.fileName = '';
            vm.$refs.importForm.resetFields();
            vm.$refs['importCtableDevice'].clearSelection();
        },
        // 配置文件详情
        fileDetailInfo(row) {
            var vm = this,
                params = {
                    smallCellCode: row.smallCellCode,
                };
            vm.fileDetailInfoShow = true;
            vm.fileDetailLoading = true;
            axios.post('${ctx}/task/BatchConfigurationFile/getConfigFileInfo.action', stringify(params)).then(function(res){
                var data = res.data;
                if(data) {
                    vm.fileDetailContent = data;
                }
                vm.fileDetailLoading = false;
            });
        },
	},
	mounted(){
		var vm = this;
		this.init();
	},

})

</script>
