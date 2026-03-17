<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<!DOCTYPE html>
<html>
<head>
    <style>
        #enbLicensePage .tabMainBoxCls{
            display: flex;
            flex-direction: column;
            height: 100%;
        }
        #enbLicensePage .tabMainBoxCls .taskListBoxCls{
            position: relative;
            border-radius: 5px;
            overflow: hidden;
            border-bottom: 1px solid #E9E9E9
        }
        #enbLicensePage .tabMainBoxCls .resultListBoxCls{
            position: relative;
            height: 50%;
            overflow: hidden;
            border-radius: 5px;
        }
        #enbLicensePage .tabMainBoxCls .dividingLineBoxCls{
            height: 1%;
            background-color: #f6f7fb;
        }
        #enbLicensePage .resultListBoxCls .logFileTitle{
            display: flex;
            align-items: center;
            font-size: 14px;
            padding-left: 20px;
            font-weight: 550;
            color:#333333;
        }
        #enbLicensePage .switchBoxCls-ctn {
            display: inline-block;
            position: relative;
            cursor: pointer;
        }
        #enbLicensePage .switchBoxCls-ctn::before {
            content: '';
            position: absolute;
            top: 0;
            right: 0;
            bottom: 0;
            left: 0;
            z-index: 100;
        }
        #enbLicensePage .el-tabs .el-tabs__header{
            border-bottom: 1px solid #DFE2EE;
        }
        #enbLicensePage .el-date-editor .el-range__close-icon {
            line-height:20px;
        }
        #enbLicensePage .taskListBoxCls .el-ctable-toolbar{
            padding: 0px !important;
        }
        .licenseFileInfoCls .textareaBox .el-textarea {
            width: 100%;height: 100%; border: 0;
        }
        .licenseFileInfoCls .textareaBox .el-textarea__inner {border: 0;width: 100%;height: 97%;resize: none; font-size: 12px; color: rgba(0, 0, 0, 0.8) !important;}
        #enbLicensePage .rightImportBox .el-form-item{
            margin-bottom: 20px;
        }
        #enbLicensePage .rightImportBox .el-form-item .el-select>.el-input{
            width: 100%;
        }
        #enbLicensePage .licIconItem .el-icon{
            font-size:18px;
            vertical-align:bottom;
            margin-right:5px;
        }
        #enbLicensePage .licIconItem .el-icon-status-execute-success:before{
            color:#67D972;
        }
        #enbLicensePage .licIconItem .el-icon-status-execute-failure:before{
            color:#E88282;
        }
        #enbLicensePage .connauto{
            margin:0 auto;
        }
  	</style>
</head>
<body>
	<!-- enb License -->
	<div id="enbLicensePage">
        <div class='commonFlex' style='width: 100%; height: 100%;'>
            <div  class="container commonWarp leftWarp" style="position: relative;">
                <!-- 主页面区域 -- tab页 -->
                <el-tabs v-model="activeName" @tab-click='tabClick' style="height: 100%;">
                    <!-- 4G  Basic License -->
                    <el-tab-pane label="Basic License" name="basic">
                        <div class="tabMainBoxCls">
                            <div class="taskListBoxCls" :style="{'height': '49%' }">
                                <!-- Basic License 操作按钮 -->
                                <div v-if="optWriteShow" class="newIconBoxCls-bt" style="right:100px;top:5px;" @click="batchInputBasicLicenseTask" tip="<%=rb.getString("PiLiangShuRu")%>">
                                    <span class='el-icon el-icon-batchInput'></span>
                                </div>
                                <div v-if="optWriteShow" class="newIconBoxCls-bt" @click="importBasicLicense" style="right:60px;top:5px;" tip="<%=rb.getString("DaoRu")%>">
                                    <span class='el-icon el-icon-operation-import'></span>
                                </div>
                                <div class="newIconBoxCls-bt" @click="exportBasicLicenseTask" style="right:20px;top:5px;" tip="<%=rb.getString("DaoChu")%>">
                                    <span class='el-icon el-icon-operation-export'></span>
                                </div>
                                <!-- 表格组件 -->
                                <el-ctable 
                                    ref="enbBasicLicenseTaskList" 
                                    id="enbBasicLicenseTaskList" 
                                    :url="enbBasicLicenseTaskUrl" 
                                    :query-params="enbBasicLicenseQueryParams" 
                                    :time="6"
                                    row-key="serial_number"
                                    @selection-change='enbBasicLicenseTaskBatchSelect'
                                >
                                    <template slot="toolbar">
                                        <div class='toolbarHeadBtnBoxCls'>
                                            <div v-show="optWriteShow" class="selectBlukBoxCls">
                                                <div class="selectMain">
                                                    <div class="bulkSelectBtnBoxCls"  @click="openBulkSelectTable">
                                                        <span class="el-icon-selected el-icon"></span>
                                                        <span class="bulkSelectNumBoxCls">( {{enbBasicLicenseTaskSelection.length}} )</span>
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
                                                                    :data="enbBasicLicenseTaskSelection"
                                                                    :showHeader="false"
                                                                    :rownumber="false"
                                                                    :front-pagination="true"
                                                                    row-key="serial_number"
                                                                    height="270px" pagination="true" >
                                                                    <el-table-column prop="serial_number" v-if="false"></el-table-column>
                                                                    <el-table-column width="588">
                                                                        <template slot-scope="scope" >
                                                                            <div class="tableItemCls">
                                                                                <span>{{scope.row.serial_number}}</span>
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
                                            <div v-show="optWriteShow" :class="enbBasicLicenseTaskSelection.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="sendLicense('','batch')">
                                                <span class="el-icon el-icon-operation-issued"></span>
                                                <span>Send License</span>
                                            </div>
                                            <div v-show="optWriteShow" :class="enbBasicLicenseTaskSelection.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="downloadLicenseFile('','batch')">
                                                <span class="el-icon el-icon-operation-download"></span>
                                                <span><%=rb.getString("XiaZai")%></span>
                                            </div>
                                            <div v-show="optWriteShow" :class="enbBasicLicenseTaskSelection.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="deleteBasicLicenseTask('','batch')">
                                                <span class="el-icon el-icon-operation-delete"></span>
                                                <span><%=rb.getString("ShanChu")%></span>
                                            </div>
                                        </div>
                                        <div class='commonQuery' style='display: flex;align-items: center;height:45px;'>
                                            <el-query type="normal" @query="query" placeholder='<%=rb.getString("XiaoZhanBianMa")%>' style="margin-right: 10px;"></el-query>
                                            <el-popfilter
                                                style="margin-right: 10px;"
                                                label='<%=rb.getString("ZaiXianZhuangTai") %>'
                                                v-model="enbBasicLicenseQueryParams.connection_status"
                                                type="single"
                                                :list="connectionStatusList">
                                            </el-popfilter>
                                            <el-popfilter
                                                style="margin-right: 10px;"
                                                label='<%=rb.getString("ZhuangTai")%>'
                                                v-model="enbBasicLicenseQueryParams.execute_status"
                                                type="single"
                                                :list="taskStatusList">
                                            </el-popfilter>
                                            <el-popfilter
                                                style="margin-right: 10px;"
                                                label='<%=rb.getString("ChanPinLeiXingBiaoZhi")%>'
                                                v-model="enbBasicLicenseQueryParams.product"
                                                type="single"
                                                :list="headerProductList.map(item=>{return {label:item.name,value:item.value}})">
                                            </el-popfilter>
                                            <div class="pop-filter-clear" style="margin: 0 5px;" 
                                                @click="resetQuery">
                                                <%=rb.getString("QingKongShaiXuan")%>
                                            </div>
                                        </div>
                                    </template>					
                                    <!-- 列表columns -->
                                    <el-table-column width="50" type="selection" prop="ck" :reserve-selection="true" v-if="optWriteShow"></el-table-column>
                                    <el-table-column prop="op" label=" " width="45" align="center">
                                        <template slot-scope="scope">
                                            <div class="el-icon el-icon-operation-more" v-clickoutside="hideMenus" @click="optClick(scope.row,event)"></div>
                                        </template>
                                    </el-table-column>
                                    <el-table-column prop="connection_status" width="50">
                                        <template slot-scope="scope">
                                            <div v-html="licenseConnectionStatusFormatter(scope.row.connection_status,scope.row)"></div>
                                        </template>
                                    </el-table-column>
                                    <el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="180" prop="serial_number" sortable></el-table-column>
                                    <el-table-column label='<%=rb.getString("ChanPinLeiXingBiaoZhi")%>' min-width="140" prop="product"></el-table-column>
                                    <el-table-column label='<%=rb.getString("LicenseWenJian")%>' min-width="200" prop="file_name"></el-table-column>
                                    <el-table-column label='<%=rb.getString("ShangChuanShiJian")%>' min-width="220" prop="upload_time" sortable></el-table-column>
                                    <el-table-column label='<%=rb.getString("ZhuangTai")%>' min-width="180" prop="execute_status">
                                        <template slot-scope="scope">
                                            <div v-if="scope.row.execute_status === 0" class='licIconItem'><span class='el-icon el-icon-status-unexecuted'></span><%=rb.getString("WeiZhiXing")%></div>
                                            <div v-if="scope.row.execute_status === 1" class='licIconItem'><span class='el-icon el-icon-status-executing'></span><%=rb.getString("ZhengZaiZhiXing")%></div>
                                            <div v-if="scope.row.execute_status === 2" class='licIconItem'><span class='el-icon el-icon-status-execute-success'></span><%=rb.getString("ZhiXingChengGong")%></div>
                                            <div v-if="scope.row.execute_status === 3" class='licIconItem'><span class='el-icon el-icon-status-execute-failure'></span><%=rb.getString("ZhiXingShiBai")%></div>
                                        </template>
                                    </el-table-column>
                                </el-ctable>
                                
                            </div>
                            <div class="dividingLineBoxCls"></div>
                            <div class="resultListBoxCls">
                                <!-- 操作按钮 -->
                                <div v-if="optWriteShow" class="newIconBoxCls-bt" style="right: 60px;top: 10px;" @click="clearBasicLicenseTaskLog" tip="<%=rb.getString("QingKong")%>">
                                    <span class='el-icon el-icon-operation-delete'></span>
                                </div>
                                <div class="newIconBoxCls-bt" style="right: 20px;top: 10px;" @click="exportEnbBasicLicenseTaskLog" tip="<%=rb.getString("DaoChu")%>">
                                    <span class='el-icon el-icon-operation-export'></span>
                                </div>
                                <el-ctable ref="enbBasicLicenseTaskLogList" id="enbBasicLicenseTaskLogTable" :url="enbBasicLicenseLogUrl" :query-params="enbBasicLicenseLogParams" :time="6">
                                    <!-- 头部 -->
                                    <template slot="toolbar">
                                        <div class='commonQuery' style='display: flex;align-items: center;'>
                                            <span style="font-size:14px;;margin-left: 20px;font-weight:bold;"><%=rb.getString("RiZhi")%></span>
                                            <el-query type="normal" @query="queryBasicLicenseLog" placeholder='<%=rb.getString("XiaoZhanBianMa")%>'></el-query>
                                        </div>
                                    </template>
                                    <el-table-column prop="serial_number" label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="180"></el-table-column>
                                    <el-table-column prop="start_time" label='<%=rb.getString("KaiShiShiJian")%>' min-width="160" sortable></el-table-column>
                                    <el-table-column prop="task_progress" label='<%=rb.getString("RenWuJinDu")%>' min-width="180">
                                        <template slot-scope="scope">
                                            <div v-if="scope.row.task_progress === 0" class='licIconItem'><%=rb.getString("DengDaiZhiXing")%></div>
                                            <div v-if="scope.row.task_progress === 1" class='licIconItem'><%=rb.getString("JinXingZhong")%></div>
                                            <div v-if="scope.row.task_progress === 2" class='licIconItem'><%=rb.getString("YiWanCheng")%></div>
                                        </template>
                                    </el-table-column>
                                    <el-table-column prop="result" label='<%=rb.getString("JieGuo")%>' min-width="200"></el-table-column>
                                </el-ctable>
                            </div>
                        </div>
                    </el-tab-pane>
                    <!-- 1588 License -->
                    <el-tab-pane label='1588 License' name="1588License" v-if="!isCloudShow">
                        <!-- 1588操作按钮 -->
                        <div v-if="optWriteShow" class="newIconBoxCls-bt" style="right:100px;top:10px;" @click="import1588LicenseTask" tip="<%=rb.getString("DaoRuWenJian")%>">
                            <span class='el-icon el-icon-operation-import'></span>
                        </div>
                        <div v-if="optWriteShow" class="newIconBoxCls-bt" style="right:60px;top:10px;" @click="add1588LicenseTask" tip="<%=rb.getString("XinJianRenWu")%>">
                            <span class='el-icon el-icon-plus'></span>
                        </div>
                        <div v-if="optWriteShow" class="newIconBoxCls-bt" style="right:20px;top:10px;" @click="open1588LicenseTaskList" tip="<%=rb.getString("RenWuLieBiao")%>">
                            <span class='el-icon el-icon-circle-List'></span>
                        </div>
                        <el-ctable ref="enb1588LicenseTableList" id="enb1588LicenseTableList" :url='enb1588LicenseTableUrl' :query-params="queryParams1588License" pagination="true">
                            <template slot="toolbar">
                                <div class="searchCon" style="margin-left:18px;">
                                    <el-query type="normal" @query="query1588License" placeholder='1588 License File' style="margin-right: 10px;"></el-query>
                                </div>
                            </template>
                            <el-table-column label='' width="45" class-name="no-text-tips" align="center">
                                <template slot-scope="scope">
                                        <div class="el-icon el-icon-operation-more" @click="optClick1588License(scope.row,event)" v-clickoutside="handerClose1588License" style="cursor: pointer;"></div>
                                  </template>
                            </el-table-column>
                            <el-table-column prop='file_name' label='1588 License File' min-width="300"></el-table-column>
                            <el-table-column prop='mac_range' label='<%=rb.getString("MACDiZhi")%>' min-width="160" show-overflow-tooltip="true"></el-table-column>
                            <el-table-column prop='file_size' label='<%=rb.getString("WenJianDaXiao")%>' min-width="120"></el-table-column>
                            <el-table-column prop='uploader' label='<%=rb.getString("ShangChuanRen")%>' min-width="120"></el-table-column>
                            <el-table-column prop='upload_time' label='<%=rb.getString("ShangChuanShiJian")%>' min-width="150"></el-table-column>
                            <el-table-column prop='description' label='<%=rb.getString("MiaoShu")%>' min-width="160"></el-table-column>
                        </el-ctable>
                        
                    </el-tab-pane>
                </el-tabs>
                <!-- 右键菜单 -->
                <el-cmenu ref="menu"  @click="menuClick" :data="menus"></el-cmenu>
                <el-cmenu ref="menu1588License" :data="menus1588License" @click="menuClick1588License"></el-cmenu>
                <!-- 新建，详情跳转 -->
                <el-slide ref="enbLicenseSlide" :title="slideTitle" method='get'
                    :url="slideUrl"  
                    :footer="slideFooter" 
                    :header='slideHeader' 
                    :position="slidePosition"
                    :height="slideHeight" :modal='modal' 
                    :width="slideWidth" 
                    @ok='saveEnbLicenseSlide'
                    @cancel='cancelEnbLicenseSlide' 
                    :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'">
                    
                </el-slide>
            </div>
            <!-- 导入 Basic License 模态框 -->
            <div class='rightWarp rightImportBox' style='flex: 0 1 360px;' v-show="rightBoxShow == 'importBasicLicense'">
                <div class='rightWarpLayer'>
                    <div class='rightBoxHeaderHasTip'>
                        <div class='headerText'>
                            <span><%=rb.getString("DaoRu")%></span>
                            <span class='closeIconBox' @click='closeImportBasicLicense'><i class='el-icon el-icon-close'></i></span>
                        </div>
                    </div>
                    <div class='rightWarpLayerContent'>
                        <el-form label-position="top" ref="importFormBasicLicense" :model='importFormBasicLicense' :rules='importRulesBasicLicense' style='margin: 20px;'>     		     			            
                            <el-form-item label='<%=rb.getString("ChanPinLeiXingBiaoZhi") %>' prop="productType">
                                <el-select v-model="importFormBasicLicense.productType" style="width: 100%;">
                                    <el-option v-for="item in rightProductList" :label="item.name" :value="item.value"></el-option>
                                </el-select>
                            </el-form-item>
                            <el-form-item prop="fileName">
                                <template slot='label'>
                                    <div class='commonFlex'>
                                        <span class='commonSize14'><%=rb.getString("DaoRuWenJian")%></span>
                                        <span class='commonNotes12'> ( <%=rb.getString("LicenseGeShi")%> )</span>
                                    </div>
                                </template>
                                <el-upload 
                                    ref="uploadBasicLicense"
                                    :multiple="true" 
                                    :on-change="fileChangeBasicLicense"  
                                    :show-file-list=false 	                  		
                                    :action="importFormBasicLicense.uploadFileUrl" 
                                    :data="importFormBasicLicense" 
                                    :file-list="importFormBasicLicense.fileList" 
                                    name="uploadFile" 
                                    accept=".lic"
                                    :auto-upload="false">
                                    <el-input :readonly="true" style="width: 100%;" v-model="importFormBasicLicense.fileName" placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>'>
                                        <a slot="append" class="el-icon el-icon-operation-import importBox" @click="fileSelectBasicLicense"></a>
                                    </el-input>								
                                    <a slot="trigger" ref="file_up"></a>
                                </el-upload>	
                            </el-form-item> 
                        </el-form>
                    </div>
                    <div class='commonFlex commonBorderTop commonFormFotter'>
                        <div>
                            <el-button type="primary" @click="uploadSubmitBasicLicense" :disabled='importLoading'><%=rb.getString("QueDing")%></el-button>
                            <el-button @click="closeImportBasicLicense"><%=rb.getString("QuXiao")%></el-button>
                        </div>
                    </div>
                </div>
            </div>
            <!-- 导入 1588 License 模态框 -->
            <div class='rightWarp rightImportBox' style='flex: 0 1 360px;' v-show="rightBoxShow == 'import1588License'">
                <div class='rightWarpLayer'>
                    <div class='rightBoxHeaderHasTip'>
                        <div class='headerText'>
                            <span><%=rb.getString("DaoRu")%></span>
                            <span class='closeIconBox' @click='closeImport1588License'><i class='el-icon el-icon-close'></i></span>
                        </div>
                    </div>
                    <div class='rightWarpLayerContent'>
                        <el-form label-position="top" ref="importForm1588License" :model='importForm1588License' :rules='importRules1588License' style='margin: 20px;'>     		     			            
                            <el-form-item prop="fileName">
                                <template slot='label'>
                                    <div class='commonFlex'>
                                        <span class='commonSize14'><%=rb.getString("DaoRuWenJian")%></span>
                                        <span class='commonNotes12'> ( <%=rb.getString("LicenseGeShi")%> )</span>
                                    </div>
                                </template>
                                <el-upload 
                                    ref="upload1588License"
                                    :on-change="fileChange1588License"  
                                    :show-file-list=false 	                  		
                                    :action="importForm1588License.uploadFileUrl" 
                                    :data="importForm1588License" 
                                    :file-list="importForm1588License.fileList" 
                                    name="uploadFile" 
                                    accept=".lic"
                                    :auto-upload="false">
                                    <el-input :readonly="true" style="width: 100%;" v-model="importForm1588License.fileName" placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>'>
                                        <a slot="append" class="el-icon el-icon-operation-import importBox" @click="fileSelect1588License"></a>
                                    </el-input>								
                                    <a slot="trigger" ref="file_up1588License"></a>
                                </el-upload>	
                            </el-form-item> 
                            <el-form-item label='<%=rb.getString("MiaoShu") %>' prop="desc">
                                <el-input v-model='importForm1588License.desc' type='textarea' :rows='4' style='width:100%;'></el-input>
                            </el-form-item>
                        </el-form>
                    </div>
                    <div class='commonFlex commonBorderTop commonFormFotter'>
                        <div>
                            <el-button type="primary" @click="uploadSubmit1588License" :disabled='importLoading'><%=rb.getString("QueDing")%></el-button>
                            <el-button @click="closeImport1588License"><%=rb.getString("QuXiao")%></el-button>
                        </div>
                    </div>
                </div>
            </div>
        </div>
        <!-- 信息 -->
        <el-dialog custom-class="licenseFileInfoCls" title='<%=rb.getString("WenJianXinXi")%>' :visible.sync="licenseFileInfoDialogShow" width="40%" :close-on-click-modal="false">
            <div style="height:60%;width: 100%;padding: 10px;" class='textareaBox'>
                <el-input type="textarea" v-model="licenseFileContent" class="border border-box"></el-input>
           </div>
        </el-dialog>
        <!-- 批量选中 -->
        <el-dialog class='dialogStyle' title='<%=rb.getString("PiLiangShuRu")%>' width='630px' :visible.sync='batchInputDialogShow' :append-to-body="true" :close-on-click-modal="false" @close='closeBatchSn'>
            <el-form ref='batchInputForm' :rules='batchInputRules' :model='batchInputForm' label-position="top">
                <div>
                    <label><%=rb.getString("XiaoZhanBianMa")%></label>
                    <el-form-item prop='serialNumber' style="margin-bottom:22px;">
                        <el-input v-model='batchInputForm.serialNumber' type='textarea' :rows="4" style='margin-top:5px;'></el-input>
                    </el-form-item>
                    <p style='display:flex;color:#BBB'><span style="font-size:12px;"><%=rb.getString("eNBZhuCeTiShiWenZi") %></span></p>
                </div>
                <div style='margin-top:45px;'>
                    <el-button @click='saveBatchSn' type="primary"><%=rb.getString("QueDing")%></el-button>
                    <el-button @click='closeBatchSn'><%=rb.getString("QuXiao")%></el-button>
                </div>
            </el-form>
        </el-dialog>
	</div>
	<script>
	  	var enbLicenseVue = new Vue({
		  	el: '#enbLicensePage',
			data () {
                var vm = this,
                    validateProductType = function(rule,value,callback){			  
                        if(value === '' || value === null || value === undefined) {
                            callback(new Error('<%=rb.getString("QingXuanZeChanPinLeiXing")%>'));
                        }else{
                            callback();
                        }
                    },
                    validateFilesName = function(rule,value,callback) {
                        if( value === '' || value === null || value === undefined) {
                            callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
                        }else {
                            callback();
                        }
                    },
                    validatorNum = (rule,value,callback) => {
                        var serialNumber = value, temp = /^(\d|[a-zA-Z]|-|\s){1,30}$/,
                            list = serialNumber.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ 
                                return item.length > 0;
                            });
                
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
                    };
				return {
	          	 	enbBasicLicenseTaskUrl: '${ctx}/cell/license/getAllLicenseInfoData.action',
                    activeName:'basic',
                    connectionStatusList:[
                        {label:'<%=rb.getString("QuanBu")%>',value:''},
                        {label:'<%=rb.getString("LianJieZhengChang")%>',value:"1"},
                        {label:'<%=rb.getString("LianJieDuanKai")%>',value:"0"},
                        {label:'<%=rb.getString("TongBuZhong")%>',value:"3"},
                        {label:'<%=rb.getString("TongBuShiBai")%>',value:"2"},
                        {label:'<%=rb.getString("ChuShiHuaZhong")%>',value:"4"},
                        {label:'<%=rb.getString("YuanTongBuZhong")%>',value:"5"},
                        {label:'<%=rb.getString("YuanTongBuWanCheng")%>',value:"6"}
                    ],
                    taskStatusList: [
                        {label:'<%=rb.getString("QuanBu")%>',value:""},
                        {label:'<%=rb.getString("WeiZhiXing")%>',value:"0"},
                        {label:'<%=rb.getString("ZhengZaiZhiXing")%>',value:"1"},
                        {label:'<%=rb.getString("ZhiXingChengGong")%>',value:"2"},
                        {label:'<%=rb.getString("ZhiXingShiBai")%>',value:"3"}
                    ],
                    headerProductList:[],
                    rightProductList:[],
                    enbBasicLicenseTaskSelection:[],
                    bulkSelectShow:false,
                    enbBasicLicenseQueryParams: {
			            timeZone: timeZone,
                        search_text: '',
                        execute_status: '',
                        connection_status: '',
                        product: ''
		            },
                    
		            enbBasicLicenseLogUrl: '${ctx}/cell/license/getLicenseRecordData.action',
                    
                    enbBasicLicenseLogParams: {
		          		search_text: '',
			            timeZone: timeZone,
		          	},
	         		menus: [],
	         		rowData: [],
	         		//slide
                    slideUrl: '',
	         		slideTitle:'',
	         		slideHeader:'',
	        	    slideFooter:'',
	        	    slidePosition:'',
	        	    slideHeight:'',
	        	    slideWidth:'',
	        	    operateType:'',
                    licenseFileInfoDialogShow: false,
                    licenseFileContent:'',
                    rightBoxShow: '',
                    importLoading: false,
                    importFormBasicLicense:{
                        productType: '',
                        fileList: [],
                        fileName: '',
                        uploadFileUrl: ''
                    },
                    importRulesBasicLicense:{
                        fileName:[ {validator: validateFilesName}],
                        productType: [{validator: validateProductType}],
                    },
                    batchInputDialogShow: false,
                    batchInputForm:{
                        serialNumber:'',
                    },
                    batchInputRules:{
                        serialNumber:[
                            {validator:validatorNum,trigger:'change'}
                        ]
                    },
                    // 1588 License 参数
                    menus1588License: [],
                    operType: '',
                    enb1588LicenseRowData: {},
                    enb1588LicenseTableUrl:'${ctx}/cell/1588License/getFileInfoList.action',
                    queryParams1588License:{
                        time_zone:timeZone,
                        search_text:''
                    },
                    importForm1588License:{
                        desc: '',
                        fileList: [],
                        fileName: '',
                        uploadFileUrl: ''
                    },
                    importRules1588License:{
                        fileName:[ {validator: validateFilesName}],
                    },
		      	}
		    },
            computed:{
                optWriteShow() {
                    var vm = this,
                        deviceType = 'eNB'
                        codes = {
                            'eNB':'CODE_ENB_LICENSE',
                            'gNB':'CODE_GNB_LICENSE',
                        };
        
                    return writableMap[codes[deviceType]] == true;
                },
                isCloudShow() {
                    return isCloud == 'true';
                }
            },
	    	methods: {
                // 初始化数据
                init(){
                    var vm = this;
                    vm.getProductList();
                },
                // enb basic License 模糊搜索
		        query(text) {
                    var vm = this;
                    vm.enbBasicLicenseQueryParams.search_text = text;
                },
                // 查询重置
                resetQuery(){
                    var vm = this,
                        params = {
                            execute_status: '',
                            connection_status: '',
                            product: ''
                        };
                    Object.assign(vm.enbBasicLicenseQueryParams,params);
                },
                // enb basic License Log 模糊搜索
                queryBasicLicenseLog(val){
                    var vm = this;
                    vm.enbBasicLicenseLogParams.search_text = val;
                },
                // tab 切换事件
                tabClick(tab){
                    var vm = this;
                    vm.rightBoxShow = '';
                },
                // 获取产品类型下拉数据
                getProductList(){
                    var vm = this;
                    axios.post("${ctx}/task/upgrade/getProductType.action",stringify({
                        type: 'all'
                    })).then(function(res){
                        var data = res.data ? res.data : [];
                        let productTypeList = [];
                        let noAllList = [];
                        let allList = [{name: '<%=rb.getString("QuanBu")%>',value:''}];
                        data.map((item)=>{
                            if(item.value.includes("CR-B4860")){
                                if(item.value == 'CR-B4860/BU'){
                                    noAllList.push(item);
                                    allList.push(item);
                                }
                            }else{
                                noAllList.push(item);
                                allList.push(item);
                            }
                        })
                        vm.rightProductList = noAllList;
                        vm.headerProductList = allList;		
                    }); 
                },
                /**
                * 选择的批量数据
                * @param selection:传入批量数据对象
                */
                enbBasicLicenseTaskBatchSelect(selection){
                    var vm = this;
	    	        vm.enbBasicLicenseTaskSelection = selection;
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

                    vm.$refs.enbBasicLicenseTaskList.clearSelection();
                },
                // 设备已选表格 单个删除事件
                delBulkSelected(rows){
                    var vm = this,
                        deviceType = 'eNB',
                        codes = {
                            'eNB':'serial_number',
                            'gNB':'serial_number',
                        },
                        tabs = 'enbBasicLicenseTaskList',
                        rowKey = codes[deviceType];

                    vm.enbBasicLicenseTaskSelection = vm.enbBasicLicenseTaskSelection.filter((items)=>{
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
				optClick(row,ev){
			    	var vm = this, sendFlag = false, terminateFlag = false, delFlag = false, 
                        executeStatus = row.execute_status;
			    	
	    		    vm.rowData = row;
	    		  	
                    if(executeStatus === 2 || executeStatus === 3){
                        terminateFlag = true;
                    }
                    if(executeStatus === 0 || executeStatus === 2 || executeStatus === 3){
                        delFlag = false
                        sendFlag = false
                    }
                    if(executeStatus === 1){
                        delFlag = true
                        sendFlag = true
                    }
    		    	vm.menus= [
    			          {label:'Send License',cls:"el-icon el-icon-operation-issued CODE_ENB_LICENSE hidden",code:'send',disable: sendFlag},
    			          {label:'<%=rb.getString("ZhongZhiRenWu")%>',cls:"el-icon el-icon-operation-terminate CODE_ENB_LICENSE hidden",code:'terminate', disable:terminateFlag},
    			          {label:'<%=rb.getString("ChaKan")%>',cls:"el-icon el-icon-operation-view",code:'view'},
    			          {label:'<%=rb.getString("XiaZai")%>',cls:"el-icon el-icon-operation-download",code:'download'},
    			          {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_ENB_LICENSE hidden",code:'del',disable:delFlag}
    			    ]
    		    	this.$nextTick(function(){
    		    		document.body.click();
	    		    	vm.$refs.menu.show(ev);
    		    	});
		        },
                // 菜单点击事件
				menuClick(ev) {
			        var vm = this,
		              	codes = {
		    	    		send: vm.sendLicense,
		    	    		terminate: vm.terminateTrace,
                            view: vm.viewLicenseFile,
		    	    		download: vm.downloadLicenseFile,
			    	    	del: vm.deleteBasicLicenseTask,
		              	};
		
			        if(codes[ev.code]){
			    		codes[ev.code](this.rowData,'single');
                    
			    	}
				},
                // 操作菜单关闭
                hideMenus() {
                    var vm = this;
                    vm.$refs.menu.hide()
                },
				// 下发License
				sendLicense(row,type) {
          			var vm = this,
                        urls = '${ctx}/cell/license/addLicenseTask.action',
                        confirmMsg = '',
                        params = {
                            timeZone : timeZone
                        };
                    if(type == 'single'){
                        
                        if(row.product == 'RTS' || row.product == 'RTD' || row.product == 'QRTB' ){
                            confirmMsg = '<%=rb.getString("PeiZhiLicenseXuYaoShouDongChongQiJiZhan")%>';
                        }else{
                            confirmMsg = '<%=rb.getString("PeiZhiLicenseXuYaoChongQiJiZhan")%>';
                        }
                        params.sns = row.serial_number;
                    }else{
                        confirmMsg = '<%=rb.getString("PiLiangPeiZhiLicenseXuYaoChongQiJiZhan")%>';
                        let arrList = [];
                        vm.enbBasicLicenseTaskSelection.map((item)=>{
                            arrList.push(item.serial_number)
                        })
                        params.sns = arrList.join(',');
                    }
			        vm.$confirm( confirmMsg,'<%=rb.getString("QueRen")%>',{
			            customClass:'warningConfirm',
			            confirmButtonText:'<%=rb.getString("QueDing")%>',
			            cancelButtonText:'<%=rb.getString("QuXiao")%>',
			            type:'warning',
			            closeOnClickModal:false
		          	}).then(() => {
			            axios.post(urls,stringify(params)).then(function(response){
							var data = response.data;
				            vm.$refs.enbBasicLicenseTaskList.refresh();
                            vm.$refs.enbBasicLicenseTaskLogList.refresh();
                            if(type !== 'single'){
                                vm.$refs.enbBasicLicenseTaskList.clearSelection();
                            }
			            }).catch(function(error){})
		          	}).catch(function(error){})
        		},
        		//终止
        		terminateTrace(row) {
		        	var vm = this,
                        params = {
                            timeZone : timeZone,
                            sns: row.serial_number
                        };
			    	    	
		          	axios.post('${ctx}/cell/license/terminateUpgradeTask.action',stringify(params)).then(function(response){
		            	var data = response.data;
			            if(data["success"]){
			            	vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type: 'success'
							});
			                vm.$refs.enbBasicLicenseTaskList.refresh()
			            }else{
			                vm.$message.error(data["message"])
			            }
		          	})
		        },
                // 查看License文件
                viewLicenseFile(row){
			    	var vm = this,
                        params = {
                            fileName: row.file_name,
                        };
                    
                    vm.licenseFileContent = '';
                    axios.post('${ctx}/cell/license/viewLicenseFileContent.action',stringify(params)).then(function(response){
                        var data = response.data;
                        if(data["success"]){
                            if(data.message == ""){
                                vm.$message.error('<%=rb.getString("WenJianBuCunZai")%>');
                            }else{
                                vm.licenseFileInfoDialogShow = true;
                                vm.licenseFileContent = data.message;
                            }
                        }else{
                            vm.$message.error(data["message"])
                        }
                    })
			    	
			    },
		      	// 下载License文件
                downloadLicenseFile(row,type){
					var vm = this,
                        params = {
                            fileNames: ''
                        };
                    if(type == 'single'){
                        params.fileNames = row.file_name;
                    }else{
                        let arrList = [];
                        vm.enbBasicLicenseTaskSelection.map((item)=>{
                            arrList.push(item.file_name)
                        })
                        params.fileNames = arrList.join(',');
                    }
					axios.post('${ctx}/cell/license/getDownloadFileNumber.action',stringify(params)).then(function(response){
		                var data = response.data;
		                if(data.length == 0){
                            vm.$message.error('<%=rb.getString("WenJianBuCunZai")%>')
		                }else{
                            exportByForm("${ctx}/cell/license/doDownloadLicenseFile.action",{
                                fileNames: params.fileNames
                            });
		                }
		            })
				},
                // 删除 Basic License 任务
		        deleteBasicLicenseTask(row,type) {
		        	var vm = this,
                        urls = '${ctx}/cell/license/doClearLicenseFile.action',
                        params = {
                            fileNames: ''
                        };
                    if(type == 'single'){
                        params.fileNames = row.file_name;
                    }else{
                        let arrList = [];
                        let activeList = [];
                        vm.enbBasicLicenseTaskSelection.map((item)=>{
                            arrList.push(item.file_name);
                            if(item.execute_status === 1){
                                activeList.push(item)
                            }
                        })
                        params.fileNames = arrList.join(',');
                        if(activeList.length > 0){
                            vm.$message.error('<%=rb.getString("YouZhengZaiZhiXingRenWu")%>')
                            return;
                        }
                    }
		            vm.$confirm('<%=rb.getString("QueRenShanChuRenWu")%>','<%=rb.getString("QueRen")%>',{
			            customClass:'warningConfirm',
			            confirmButtonText:'<%=rb.getString("QueDing")%>',
			            cancelButtonText:'<%=rb.getString("QuXiao")%>',
			            type:'warning',
			            closeOnClickModal:false
		          	}).then(() => {
			            axios.post(urls,stringify(params)).then(function(response){
							var data = response.data;
				            if(data["success"]){
				                vm.$message({
				                	type:'success',
				                    message:'<%=rb.getString("ChengGong")%>'
				                })
                                vm.$refs.enbBasicLicenseTaskList.refresh();
                                vm.$refs.enbBasicLicenseTaskLogList.refresh();
                                vm.$refs.enbBasicLicenseTaskList.clearSelection();
				            }else{
				             	vm.$message.error(data["message"])
				            }
			            }).catch(function(error){})
		          	}).catch(function(error){})
		        },
                // 打开 批量输入 弹窗
                batchInputBasicLicenseTask(){
                    var vm = this;
                    vm.batchInputDialogShow = true;
                },
                // 批量输入 SN 保存
                saveBatchSn(){
                    var vm = this, 
                        saveBatchUrl = '${ctx}/cell/license/getCheckLicenseSN.action', 
                        params = {
                            isGnb: 0, // 0表示eNB , 1表示gNB
                            serialNumbers: '',
                        };
                    let snStr = vm.batchInputForm.serialNumber;
                    let snList = snStr.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ return item.length > 0;})
                    params.serialNumbers = snList.join(";");
                    
                    vm.$refs.batchInputForm.validate((valid) => {
                        if(valid){
                            axios.post(saveBatchUrl, stringify(params)).then((res)=>{
                                var data = res.data;
                                if(data && data.length > 0){		
                                    vm.$refs.enbBasicLicenseTaskList.appendCheckedRows(data);
                                    vm.closeBatchSn();
                                }else{
                                    vm.$message('<%=rb.getString("MeiYouKePiPeiSheBei")%>')
                                }
                            })
                        }
                    })
                },
                closeBatchSn(){
                    var vm = this;
                    vm.batchInputDialogShow = false;
			        vm.$refs.batchInputForm.resetFields();
                },
                // 导入 Basic License
                importBasicLicense(){
                    var vm = this;
                    vm.rightBoxShow = 'importBasicLicense';
                    vm.getProductList();
                    vm.importLoading = false;
                },
                /**
                * 选择文件后，校验格式，并赋值页面显示 
                * @param file{object}   文件信息
                * @param fileList{Array}  文件列表
                */ 
                fileChangeBasicLicense(file,fileList){ 
                    var vm = this,
                        fileIndex = file.name.lastIndexOf("."),
                        fileType = file.name.substr(fileIndex + 1, file.name.length);
                    if(['lic'].indexOf(fileType.toLowerCase()) === -1){
                        return false;
                    }else{
                        let arrList = [];
                        let uploadFileList = [];
                        if(fileList && fileList.length > 0){
                            fileList.map((item)=>{
                                arrList.push(item.name);
                                uploadFileList.push(item.raw)
                            })
                        }
                        vm.importFormBasicLicense.fileName = arrList.join(',');
                        vm.importFormBasicLicense.fileList = uploadFileList;
                    }
                },
                // 选择文件
                fileSelectBasicLicense(){  
                    var vm = this;
                    vm.importFormBasicLicense.fileName = '';
                    vm.importFormBasicLicense.fileList = [];
                    vm.$refs.uploadBasicLicense.clearFiles();
                    vm.$refs['file_up'].click();
                },
                //确定导入
                uploadSubmitBasicLicense() {
                    var vm = this, 
                        fileImportArr = [], 
                        productType = vm.importFormBasicLicense.productType,
                        fileList = vm.importFormBasicLicense.fileList;

                    vm.$refs.importFormBasicLicense.validate((valid) => {
                        if(valid) {
                            var fd = new FormData(),
                                config = {
                                    headers: { 'Content-Type': 'multipart/form-data' }
                                };
                            
                            if(fileList.length > 0){
                                fileList.forEach(file =>{
                                    //只有 lic 格式可行
                                    if(file.name.substr(file.name.lastIndexOf(".")) === '.lic'){
                                        var obj = {
                                            fileName: file.name,
                                            fielSize: file.size
                                        }
                                        fileImportArr.push(obj)
                                        fd.append('uploadFile',file);
                                    }
                                })
                            }
                            fd.append('fileArr',JSON.stringify(fileImportArr));//文件名
                            fd.append('productType',productType);
                            //解决在 eNB 导入 license 界面，导入 gNB 的license后， 文件显示在 gNB的 license
                            var checkParam = {
                                fileNames: vm.importFormBasicLicense.fileName
                            }
                            vm.importLoading = true;
                            axios.post("${ctx}/cell/license/checkSnsIsExist.action?isGnb=0",stringify(checkParam)).then(function(res){
                                if(res.data["success"]){	
                                    axios.post("${ctx}/cell/license/uploadLicenseFile.action",fd,config).then(function(res){
                                        if(res.data["success"]){
                                            vm.$message.success('<%=rb.getString("ChengGong")%>');
                                            vm.$refs.enbBasicLicenseTaskList.refresh();								
                                        }else{
                                            vm.$message.error(res.data["message"]);
                                        }
                                        vm.closeImportBasicLicense();
                                    })
                                }else{
                                    vm.$message.error(res.data["message"]);
                                    //失败也将确认按钮放开
                                    vm.importLoading = false;
                                }
                            })	
                        }else{
                            return false;
                        }
                    }) 				
                },
                // 关闭导入弹出框
                closeImportBasicLicense(){
                    var vm = this;	

                    vm.importFormBasicLicense.fileName = '';
                    vm.importFormBasicLicense.fileList = [];
                    vm.importFormBasicLicense.productType = '';
                    vm.$refs.importFormBasicLicense.resetFields();
                    vm.rightBoxShow = '';
                },
                // 导出 Basic License Task
                exportBasicLicenseTask(){
                    var vm = this,
                        exportUrl = '${ctx}/cell/license/exportDeviceInfos.action',
                        params = {
                            timeZone: timeZone,
                            isGnb: 0, // 0表示eNB , 1表示gNB
                            search_text: vm.enbBasicLicenseQueryParams.search_text,
                            execute_status: vm.enbBasicLicenseQueryParams.execute_status,
                            connection_status: vm.enbBasicLicenseQueryParams.connection_status,
                            product: vm.enbBasicLicenseQueryParams.product
                        };
                    exportByForm(exportUrl,params);
                },
                // 清除 Basic License Task Log
                clearBasicLicenseTaskLog(){
                    var vm = this;
                    vm.$confirm('<%=rb.getString("QueRenQingKongSuoYouRiZhi")%>','<%=rb.getString("QueRen")%>',{
                        customClass:'warningConfirm',
                        confirmButtonText:'<%=rb.getString("QueDing")%>',
                        cancelButtonText:'<%=rb.getString("QuXiao")%>',
                        type:'warning',
                        closeOnClickModal:false
                    }).then(() => {
                        axios.post('${ctx}/cell/license/clearLicenseRecord.action').then(function(response){
                            var data = response.data;
                            if(data["success"]){
                                vm.$message({
                                    type:'success',
                                    message:'<%=rb.getString("ChengGong")%>'
                                })
                                vm.$refs.enbBasicLicenseTaskList.refresh();
                                vm.$refs.enbBasicLicenseTaskLogList.refresh();
                            }else{
                                vm.$message.error('<%=rb.getString("QingChuShiBai")%>')
                            }
                        }).catch(function(error){})
                    }).catch(function(error){})
                },
				// enb Basic License Task Log-导出
				exportEnbBasicLicenseTaskLog(){
					var vm = this,
                        exportUrl = '${ctx}/cell/license/exportLicenseRecordToCSV.action',
					    params ={
                            timeZone: timeZone,
                            search_text: vm.enbBasicLicenseLogParams.search_text
                        }
                    exportByForm(exportUrl,params);
				},

                // ---------------------------------------------------------  1588 License -------------------------------------------
                
                // 导入 1588 License
                import1588LicenseTask(){
                    var vm = this;
                    vm.rightBoxShow = 'import1588License';
                    vm.importLoading = false;
                },
                // 新增 1588 License Task
                add1588LicenseTask(){
                    var vm = this;

                    if(vm.rightBoxShow){
                        vm.rightBoxShow = '';
                    }
                    vm.operType = 'addTask';
                    vm.slideTitle = '<%=rb.getString("XinJianRenWu")%>';
                    vm.slideUrl = '${ctx}/cell/1588License/goAddTaskPage.action';
                    vm.slideHeader = true;
                    vm.slideFooter = true;
                    vm.slidePosition = 'top';
                    vm.slideHeight = '100%';
                    vm.slideWidth = '100%';
                    vm.$refs.enbLicenseSlide.showSlide(function(){
                        vm.modal = false;
                    });
                },
                // 打开 1588 License Task 列表
                open1588LicenseTaskList(){
                    var vm = this;

                    if(vm.rightBoxShow){
                        vm.rightBoxShow = '';
                    }
                    vm.operType = 'taskList';
                    vm.slideTitle='1588 License <%=rb.getString("RenWuLieBiao")%>';
                    vm.slideUrl = '${ctx}/cell/1588License/goTaskListPage.action';
                    vm.slideHeader = false;
                    vm.slideFooter = false;
                    vm.slidePosition = 'top';
                    vm.slideHeight = '100%';
                    vm.slideWidth = '100%';
                    
                    vm.$refs.enbLicenseSlide.showSlide(function(){
                        vm.modal = false;
                    });
                },
                // 查询 1588 License
                query1588License(val){
                    var vm = this;
                    vm.queryParams1588License.search_text = val;
                },
                /** 1588 License
                * 点击更多操作出现菜单
                * @param row{object}   行数据
                * @param ev{object}   event数据
                */ 
                optClick1588License(row,ev){
                    var vm = this;
                    vm.enb1588LicenseRowData = row;
                    vm.menus1588License= [
                        {label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit CODE_ENB_LICENSE hidden",code:'edit'},
                        {label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info'},
                        {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_ENB_LICENSE hidden",code:'del'}
                    ]
                    vm.$nextTick(function(){
                        document.body.click();
                        vm.$refs.menu1588License.show(ev);
                    });
                },
                // 点击页面其他地方菜单收起   1588 License
                handerClose1588License(){
                    this.$refs.menu1588License.hide();
                },
                /** 1588 License
                * 菜单点击事件
                * @param ev{object}   行数据
                */ 
                menuClick1588License(ev){
                    var vm = this,
                        codes= {
                            edit:vm.edit1588License,
                            info:vm.view1588License,
                            del:vm.del1588LicenseFile
                        }
                    if(codes[ev.code]){
                        codes[ev.code](this.enb1588LicenseRowData);
                    }
                },
                // 修改 1588 License
                edit1588License(row){
                    var vm = this;

                    vm.operType = 'editLicense';
                    vm.slideTitle='<%=rb.getString("XiuGai")%>';
                    vm.slideUrl = '${ctx}/cell/1588License/goModifyInfoPage.action';
                    vm.slideHeader = true;
                    vm.slideFooter = true;
                    vm.slidePosition = 'top';
                    vm.slideHeight = '100%';
                    vm.slideWidth = '100%';
                    vm.$refs.enbLicenseSlide.showSlide(function(){
                        vm.modal = false;
                        eventBus.$emit('edit-file',row.file_id);
                    });
                },
                // 查看 1588 License
                view1588License(row){
                    var vm = this;
                    
                    vm.operType = 'viewLicense';
                    vm.slideTitle='<%=rb.getString("XinXi")%>';
                    vm.slideUrl = '${ctx}/cell/1588License/goViewInfoPage.action';
                    vm.slideHeader = true;
                    vm.slideFooter = false;
                    vm.slidePosition = 'top';
                    vm.slideHeight = '100%';
                    vm.slideWidth = '100%';
                    
                    vm.$refs.enbLicenseSlide.showSlide(function(){
                        vm.modal = false;
                        eventBus.$emit('view-file',row.file_id)
                    });
                },
                // 删除 1588 License
                del1588LicenseFile(row){
                    var vm = this;
                    vm.$confirm('<%=rb.getString("QueRenShanChuWenJian")%>','<%=rb.getString("QueRen")%>',{
                        customClass:'warningConfirm',
                        confirmButtonText:'<%=rb.getString("QueDing")%>',
                        cancelButtonText:'<%=rb.getString("QuXiao")%>',
                        type:'warning',
                        closeOnClickModal:false
                    }).then(() => {
                        axios.post('${ctx}/cell/1588License/clearFile.action',stringify({
                            file_id : row.file_id
                        })).then(function(response){
                            var data = response.data;
                            if(data["success"]){
                                vm.$refs.enb1588LicenseTableList.refresh()
                                vm.$message({
                                    type:'success',
                                    message:'<%=rb.getString("ChengGong")%>'
                                })
                            }else{
                                vm.$message.error(data["message"])
                            }
                        }).catch(function(error){
                            
                        })
                    }).catch()
                },
                //关闭新建，详情
			    cancelEnbLicenseSlide(){
		        	var vm = this;
		        	if(vm.operType == 'addTask' || vm.operType == 'editTask'){
                        eventBus.$emit('cancel-task')
                    }else if(vm.operType == 'editLicense'){
                        eventBus.$emit('cancel-file')
                    }else if(vm.operType == 'viewLicense'){
                        vm.$refs.enbLicenseSlide.hide();
                    }else if(vm.operType == 'taskList'){
                        vm.$refs.enbLicenseSlide.hide();
                    }
        		},
		        saveEnbLicenseSlide() {
	      			var vm = this;
                    if(vm.operType == 'addTask'){
                        eventBus.$emit('save-task')
                    }else if(vm.operType == 'editLicense'){
                        eventBus.$emit('save-lic')
                    }
		        },
			    hideEnbLicenseSlide(){
					var vm = this;
			    	vm.$refs.enbLicenseSlide.hide();
                    vm.$refs.enb1588LicenseTableList.refresh()
			    },
                /**
                * 选择文件后，校验格式，并赋值页面显示 
                * @param file{object}   文件信息
                * @param fileList{Array}  文件列表
                */ 
                fileChange1588License(file,fileList){ 
                    var vm = this,
                        fileIndex = file.name.lastIndexOf("."),
                        fileType = file.name.substr(fileIndex + 1, file.name.length);
                    if(['lic'].indexOf(fileType.toLowerCase()) === -1){
                        return false;
                    }else{
                        let arrList = [];
                        let uploadFileList = [];
                        if(fileList && fileList.length > 0){
                            fileList.map((item)=>{
                                arrList.push("C:\\fakepath\\" + item.name);
                                uploadFileList.push(item.raw)
                            })
                        }
                        vm.importForm1588License.fileName = arrList.join(',');
                        vm.importForm1588License.fileList = uploadFileList;
                    }
                },
                // 选择文件
                fileSelect1588License(){  
                    var vm = this;
                    vm.importForm1588License.fileName = '';
                    vm.importForm1588License.fileList = [];
                    vm.$refs.upload1588License.clearFiles();
                    vm.$refs['file_up1588License'].click();
                },
                //确定导入
                uploadSubmit1588License() {
                    var vm = this,
                        fileList = vm.importForm1588License.fileList;
                    vm.$refs.importForm1588License.validate((valid) => {
                        if(valid) {
                            var fd = new FormData(),
                                config = {
                                    headers: { 'Content-Type': 'multipart/form-data' }
                                };
                            if(fileList.length > 0){
                                fileList.forEach(file =>{
                                    //只有 lic 格式可行
                                    if(file.name.substr(file.name.lastIndexOf(".")) === '.lic'){
                                        fd.append('file',file);
                                    }
                                })
                            }
                            fd.append('desc',vm.importForm1588License.desc);
                            vm.importLoading = true;
                            axios.post('${ctx}/cell/1588License/uploadFile.action',fd,config).then(function(res){
                                if(res.data["success"]){
                                    vm.closeImport1588License();
                                    vm.$refs.enb1588LicenseTableList.refresh();
                                }else{
                                    vm.$message.error(res.data["message"]);
                                    // 失败也将确认按钮放开
                                    vm.importLoading = false;
                                }
                            })
                        }else{
                            return false;
                        }
                    }) 				
                },
                // 关闭导入弹出框
                closeImport1588License(){
                    var vm = this;	

                    vm.importForm1588License.fileName = '';
                    vm.importForm1588License.fileList = [];
                    vm.importForm1588License.productType = '';
                    vm.$refs.importForm1588License.resetFields();
                    vm.rightBoxShow = '';
                },
                // License 连接状态格式化
                licenseConnectionStatusFormatter(value,rowData,rowIndex){
                    if("initializing" == value) {
                        return "<div class='el-icon el-loading status-init'></div>";
                    }else if("syncSourceInSync" == value) {
                        return "<div class='el-icon el-loading status-sync'></div>";
                    }else if("syncSourceInSynced" == value) {
                        return "<div class='el-icon el-loading status-synced'></div>";
                    }else if ("On" == value) {
                        return "<div class='el-icon el-icon-status-conn-on connauto' style='font-size:22px;'></div>";
                    } else if("updating" == value){
                        return "<div class='status_syncing connauto'></div>";
                    }else if ("Exception" == value) {
                        return "<div class='conn_exc connauto'></div>";
                    } else {
                        return "<div class='el-icon el-icon-status-conn-off connauto' style='font-size:22px;'></div>";
                    }
                    return value;
                }
	    	},
	    	watch:{},
			mounted() {
		        var vm = this;
                vm.init();
		        eventBus.$off("hander-cancel").$on('hander-cancel',this.hideEnbLicenseSlide);

                eventBus.$off('save-suc').$on('save-suc',this.hideEnbLicenseSlide);
                eventBus.$off('to-task-view').$on('to-task-view',this.open1588LicenseTaskList);
                eventBus.$off('hide-task').$on('hide-task',this.hideEnbLicenseSlide);
                eventBus.$off('save-file-suc').$on('save-file-suc',this.hideEnbLicenseSlide);
                eventBus.$off('hide-file').$on('hide-file',this.hideEnbLicenseSlide);
			}
		})
 	</script>
</body>
</html>