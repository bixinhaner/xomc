<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>
<%
	UserInfo user = (UserInfo) session.getAttribute(ComConstants.SESSION_KEY);
%>

<script type="text/javascript" src="${ctx}/js/element/Sortable.min.js?_=${omc_ver}"></script>
<script type="text/javascript" src="${ctx}/js/element/vuedraggable.umd.min.js?_=${omc_ver}"></script>

<style>
	#kpiMeasFilePage .monitorContentBoxCls{
		height: 100%;
		position: relative;
		display: flex;
	}
	#kpiMeasFilePage .monitorContentBoxCls >div:first-child{
		flex: 1;
		position: relative;
	}
	#kpiMeasFilePage .el-ctable-toolbar{
		padding: 0px !important;
	}
	#kpiMeasFilePage .el-ctable th>.cell {
		max-height: 25px !important;
	}
    #kpiMeasFilePage .kpiMeasFileHeadCls{
        height: 40px;
        display: flex;
        align-items: center;
        position: relative;
        font-size: 14px;
        font-weight: 550;
        padding: 0px 20px;
        color:rgba(0, 0, 0, 0.8);
        border-bottom: 1px solid #DFE2EE;
    }
    #kpiMeasFilePage .toolbarHeadBtnBoxCls{
        border-bottom: none;
    }
	#kpiMeasFilePage .el-date-editor .el-range__close-icon{
		line-height: 20px;
	}
	#kpiMeasFilePage .kpiMeasFileNameBoxCls{
		display: flex;
		align-items: center;
		flex-wrap: nowrap;
	}
	#kpiMeasFilePage .kpiMeasFileNameCls{
		overflow: hidden;
		white-space: nowrap;
		text-overflow: ellipsis;
	}
</style>
<div class="overflow-cls">
    <div id="kpiMeasFilePage" class="panelDefault overHide" style="min-width: 1200px;border:none;">
        <div class="monitorContentBoxCls">
            <el-ctable id="kpiMeasFileTable" ref="kpiMeasFileTable" 
                :url="kpiMeasFileTableUrl"  
                :limit="limitBatch"
                :row-key="'fileName'"
				:time="6"  
				style="width: 100%;"
                :query-params="queryKpiMeasFileParams" 
                @selection-change='batchSelect'>
                <!-- 高级查询 -->
                <template slot="toolbar">
                    <div class="kpiMeasFileHeadCls">
						<span style="margin-right: 10px;"><%=rb.getString("WenJian")%></span>
                        (<span style="padding:0px 5px;color:#4D84FF;font-weight: bold;">SN:{{rowDeviceData.serialNumber}}</span>)

                        <div class="headcloseBtn" style="top:6px;right:10px;" @click="closeKpiMeasFilePage">
                            <span class="el-icon el-icon-close" style='position: unset; display: block; text-align: center;'></span>
                        </div>
                    </div>
                    <div class="toolbarHeadBtnBoxCls">
                        <div v-show="optBtnShow" class="selectBlukBoxCls">
                            <div class="selectMain">
                                    <div class="bulkSelectBtnBoxCls"  @click="openBulkSelectTable">
                                    <span class="el-icon-selected el-icon"></span>
                                    <span class="bulkSelectNumBoxCls">( {{selectionData.length}} )</span>
                                </div>
                                <div class="selectTableBoxCls" style="position: absolute;top: 38px;left: 0px;" v-show="bulkSelectShow">
                                    <div class="selectBoxTitle">
                                        <span><%=rb.getString("YiXuan")%></span>
                                        <span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectTable()"></span>
                                    </div>
                                    <div class="selectBoxMain">
                                        <div class="tableInfoCls">
                                            <div class="tableInfoHeader">
                                                <div><%=rb.getString("WenJianMing")%></div>
                                                <div @click="clearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span>Clear</div>
                                            </div>
                                            <el-ctable 
                                                id="kpiMeasFileBulkSelectTable" 
                                                ref="bulkSelectTable" 
                                                :data="selectionData" 
                                                :showHeader="false"
                                                :rownumber="false"
                                                :front-pagination="true"
                                                :row-key="'fileName'"
                                                height="270px" pagination="true" >
                                                <el-table-column prop="fileId" v-if="false"></el-table-column>
                                                <el-table-column width="588">
                                                    <template slot-scope="scope" >
                                                        <div class="tableItemCls">
                                                            <span>{{scope.row.fileName}}</span>
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
                        <div v-show="optBtnShow" :class="selectionData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="batchKpiMeasFileDownload">
                            <span class="el-icon el-icon-operation-download"></span>
                            <span><%=rb.getString("XiaZai")%></span>
                        </div>
                        <div v-show="optBtnShow" :class="selectionData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="batchKpiMeasFileDelete">
                            <span class="el-icon el-icon-operation-delete"></span>
                            <span><%=rb.getString("ShanChu")%></span>
                        </div>
						<el-date-picker style='margin-left: 10px;' 
							v-model="dateValue"
							type="daterange"
							value-format="yyyy-MM-dd"
							range-separator="——"  
							@change="dateChange"
							start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
							end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
						</el-date-picker>
                    </div>
                </template>
                
                <el-table-column label='' width="50" type="selection" :reserve-selection="true" v-if="optBtnShow"></el-table-column>
                <el-table-column label='<%=rb.getString("WenJianMing")%>' min-width="200" prop="fileName" show-overflow-tooltip>
                    <template slot-scope="scope">
                        <div class="kpiMeasFileNameBoxCls">
							<div class="operation_zip" style="height: 18px; width:18px;margin-right: 5px;"></div>
                            <div class="kpiMeasFileNameCls">{{scope.row.fileName}}</div>
                        </div>
                    </template>
                </el-table-column>
                <el-table-column label='<%=rb.getString("ShangChuanShiJian") %>' width="160" prop="uploadTime" sortable></el-table-column>
            </el-ctable>
        </div>
    </div>
</div>
<script type="text/javascript">
var refreshTable;
if(window.kpiMeasFilePage) {
	try {
		window.kpiMeasFilePage.$destroy();
	}catch(e){}
}
window.kpiMeasFilePage = new Vue({
	el:'#kpiMeasFilePage',
	data(){
		return{
            rowDeviceData:'',
			kpiMeasFileTableUrl:'',
			queryKpiMeasFileParams:{
				timeZone: timeZone,
				smallCellCode:'',
                startTime:'',
                endTime:'',
			},
			selectionData:[],
			modal:false,
			dateValue:[],
			bulkSelectShow:false,
		}		
	},
	computed: {
		limitBatch(){
			return batchOperation ? '' : 1;
		},
		optBtnShow() {
			return writableMap['CODE_PERFORMANCE_MEASUREMENT'] == true;
		},
	},
	
	watch:{},
	methods:{
		init(row){
            var vm = this;
            vm.rowDeviceData = row;
			vm.queryKpiMeasFileParams.smallCellCode = row.smallCellCode;
			vm.kpiMeasFileTableUrl = '${ctx}/pm/customizemg/getPMFileList.action';
		},
		// 时间粒度查询
		dateChange(val) {
			var vm = this;
			vm.dateValue = val;
			if(val != null){
				vm.queryKpiMeasFileParams.startTime = vm.dateValue[0];
				vm.queryKpiMeasFileParams.endTime = vm.dateValue[1];
			}else{
				vm.queryKpiMeasFileParams.startTime = '';
				vm.queryKpiMeasFileParams.endTime = '';
			}
		},
		// 打开已选弹窗
		openBulkSelectTable(type){
			var vm = this;
			vm.bulkSelectShow = true
		},
		// 关闭已选弹窗
		closeBulkSelectTable(type){
			var vm = this;
			vm.bulkSelectShow = false;

			
			
		},
		// 设备已选表格 清空事件
		clearBulkSelected(){
			var vm = this;
			vm.$refs["kpiMeasFileTable"].clearSelection();
			
		},
		// 设备已选表格 单个删除事件
		delBulkSelected(rows){
			var vm = this,
				tabs = 'kpiMeasFileTable',
				rowKey = 'fileName';
			vm.selectionData = vm.selectionData.filter((items)=>{
				return items[rowKey] != rows[rowKey]
			});
			var selection = this.$refs[tabs].$refs.ctableInner.store.states.selection,
				irow= selection.filter((items)=>{
					return items[rowKey] == rows[rowKey]
				})[0];
			vm.$refs[tabs].toggleRowSelection(irow,false);
			var idx = vm.$refs[tabs].ckList.indexOf(rows[rowKey])
			vm.$refs[tabs].ckList.splice(idx,1);
		},
		/**
		 * 多选响应
		 * @param selection:选择的数据
		*/
	    batchSelect(selection){
	    	var vm = this;

	    	vm.selectionData = selection
		},
        // 测量维护文件 批量下载
        batchKpiMeasFileDownload(){
			var vm = this, 
				fileList=JSON.stringify(vm.selectionData),
				url='${ctx}/pm/customizemg/doDownloadEnbKPIFileZip.action',
				params={
					timeZone: timeZone,
					serialNumber:vm.rowDeviceData.serialNumber,
					fileList:fileList
				};
			exportByForm(url,params);
			vm.clearBulkSelected();
        },
        // 测量维护文件 批量删除
        batchKpiMeasFileDelete(){
			var vm = this, 
				fileList=JSON.stringify(vm.selectionData),
				url='${ctx}/pm/customizemg/deleteFiles.action',
				params={
					timeZone: timeZone,
					serialNumber:vm.rowDeviceData.serialNumber,
					fileList:fileList
				};
			vm.$confirm('<%=rb.getString("QueRenShanChu")%>','<%=rb.getString("QueRen")%>',{
				customClass:"warningConfirm",
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancelButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning',
				closeOnClickModal:false
			}).then(() => {
				axios.post(url,stringify(params)).then(function(res){
					var data= res.data
					if(data.success){
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type:'success',
						});
						vm.$refs.kpiMeasFileTable.refresh();
						vm.clearBulkSelected();
					}else{
						vm.$message({
							message: data.message,
							type:'error',
						});
					}
				}).catch(() => {})
			})
			
        },
        // 关闭测量维护文件页面
        closeKpiMeasFilePage(){
            var vm = this;
            kpiMeasPage.$refs.slideView.hide();
        }
	},
    mounted(){
        eventBus.$off("kpiMeas-file").$on("kpiMeas-file",this.init)
	},
})
</script>